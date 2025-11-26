package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/drata/drata-agent-cli/internal/api"
	"github.com/drata/drata-agent-cli/internal/config"
	"github.com/drata/drata-agent-cli/internal/datastore"
	"github.com/drata/drata-agent-cli/internal/osquery"
	"github.com/drata/drata-agent-cli/internal/scheduler"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the agent as a daemon with periodic syncs",
	Long: `Run the Drata Agent as a daemon that periodically syncs system information.

The daemon will run in the foreground and sync at the configured interval.
By default, syncs occur every 2 hours.

Configuration can be set via:
- Config file: $HOME/.drata-agent/config.yaml
- Environment variables: DRATA_SYNC_INTERVAL_HOURS, etc.
- Command line flags

Example:
  drata-agent daemon
  drata-agent daemon --interval 4`,
	RunE: runDaemon,
}

var syncInterval int

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.Flags().IntVarP(&syncInterval, "interval", "i", 0, "Sync interval in hours (default: 2)")
}

func runDaemon(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override sync interval if provided
	if syncInterval > 0 {
		cfg.SyncIntervalHours = syncInterval
	}

	// Initialize data store
	ds, err := datastore.New()
	if err != nil {
		return fmt.Errorf("failed to initialize data store: %w", err)
	}

	// Check if registered
	if !ds.IsRegistered() {
		return fmt.Errorf("agent is not registered. Use 'drata-agent register' first")
	}

	// Initialize osquery client
	osq, err := osquery.NewClient(cfg.OsqueryPath)
	if err != nil {
		return fmt.Errorf("failed to initialize osquery: %w", err)
	}

	// Initialize API client
	apiClient := api.NewClient(cfg, ds)

	// Create scheduler
	sched := scheduler.NewScheduler()

	// Define sync action
	syncAction := func() {
		if err := performSync(cfg, ds, osq, apiClient); err != nil {
			log.Printf("Sync error: %v", err)
		}
	}

	// Schedule periodic sync
	if err := sched.ScheduleJob("sync", cfg.SyncIntervalHours, syncAction); err != nil {
		return fmt.Errorf("failed to schedule sync: %w", err)
	}

	// Start scheduler
	sched.Start()

	fmt.Printf("Drata Agent daemon started\n")
	fmt.Printf("Version: %s\n", cfg.Version)
	fmt.Printf("Sync interval: every %d hours\n", cfg.SyncIntervalHours)
	fmt.Printf("Press Ctrl+C to stop\n")
	fmt.Println()

	// Run initial sync after a short delay
	go func() {
		time.Sleep(10 * time.Second)
		log.Println("Running initial sync...")
		syncAction()
	}()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nShutting down...")

	// Stop scheduler
	ctx := sched.Stop()
	<-ctx.Done()

	fmt.Println("Daemon stopped")
	return nil
}

func performSync(cfg *config.Config, ds *datastore.DataStore, osq *osquery.Client, apiClient *api.Client) error {
	// Check sync throttling
	if ds.GetSyncState() == datastore.SyncStateRunning {
		log.Println("Sync already in progress, skipping")
		return nil
	}

	minutesSinceLastAttempt := ds.MinutesSinceLastAttempt()
	if minutesSinceLastAttempt >= 0 && minutesSinceLastAttempt < cfg.MinMinutesBetweenSyncs {
		log.Printf("Last sync attempt was %d minutes ago, skipping (min: %d)", minutesSinceLastAttempt, cfg.MinMinutesBetweenSyncs)
		return nil
	}

	hoursSinceLastSuccess := ds.HoursSinceLastSuccess()
	if hoursSinceLastSuccess >= 0 && hoursSinceLastSuccess < cfg.MinHoursSinceLastSync {
		log.Printf("Last successful sync was %d hours ago, skipping (min: %d)", hoursSinceLastSuccess, cfg.MinHoursSinceLastSync)
		return nil
	}

	// Set sync state
	if err := ds.SetSyncState(datastore.SyncStateRunning); err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}
	if err := ds.SetLastSyncAttemptedAt(time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("failed to update last sync attempted: %w", err)
	}

	log.Println("Starting sync...")

	// Get initialization data if needed
	if !ds.IsInitDataReady() {
		log.Println("Fetching initialization data...")
		if _, err := apiClient.GetInitData(); err != nil {
			ds.SetSyncState(datastore.SyncStateError)
			return fmt.Errorf("failed to get init data: %w", err)
		}
	}

	// Collect system information
	log.Println("Collecting system information...")
	queryResult, err := osq.GetSystemInfo(cfg.Version)
	if err != nil {
		ds.SetSyncState(datastore.SyncStateError)
		return fmt.Errorf("failed to collect system info: %w", err)
	}

	// Send to Drata
	log.Println("Sending data to Drata...")
	_, err = apiClient.Sync(queryResult)
	if err != nil {
		ds.SetSyncState(datastore.SyncStateError)
		return fmt.Errorf("failed to sync: %w", err)
	}

	// Update sync state
	if err := ds.SetSyncState(datastore.SyncStateSuccess); err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}

	log.Println("✓ Sync completed successfully")
	return nil
}
