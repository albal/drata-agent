package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/drata/drata-agent-cli/internal/api"
	"github.com/drata/drata-agent-cli/internal/config"
	"github.com/drata/drata-agent-cli/internal/datastore"
	"github.com/drata/drata-agent-cli/internal/osquery"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync system information with Drata",
	Long: `Sync your system's security configuration with Drata.

This command collects read-only information about your system's security settings
and sends it to Drata for compliance monitoring.

The following information is collected:
- Operating system version
- Disk encryption status
- Firewall status
- Screen lock settings
- Installed applications
- Browser extensions
- Auto-update settings

Example:
  drata-agent sync`,
	RunE: runSync,
}

var forceSync bool
var verboseSync bool

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVarP(&forceSync, "force", "f", false, "Force sync even if recently synced")
	syncCmd.Flags().BoolVarP(&verboseSync, "verbose", "v", false, "Show verbose output including queries being executed")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
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

	// Check sync throttling (unless forced)
	if !forceSync {
		if ds.GetSyncState() == datastore.SyncStateRunning {
			return fmt.Errorf("sync is already in progress")
		}

		minutesSinceLastAttempt := ds.MinutesSinceLastAttempt()
		if minutesSinceLastAttempt >= 0 && minutesSinceLastAttempt < cfg.MinMinutesBetweenSyncs {
			return fmt.Errorf("sync was attempted %d minutes ago. Wait %d more minutes or use --force",
				minutesSinceLastAttempt, cfg.MinMinutesBetweenSyncs-minutesSinceLastAttempt)
		}

		hoursSinceLastSuccess := ds.HoursSinceLastSuccess()
		if hoursSinceLastSuccess >= 0 && hoursSinceLastSuccess < cfg.MinHoursSinceLastSync {
			fmt.Printf("Last successful sync was %d hours ago. Skipping automatic sync.\n", hoursSinceLastSuccess)
			fmt.Println("Use --force to sync anyway.")
			return nil
		}
	}

	// Initialize osquery client with verbose option
	osq, err := osquery.NewClientWithVerbose(cfg.OsqueryPath, verboseSync)
	if err != nil {
		return fmt.Errorf("failed to initialize osquery: %w", err)
	}

	if verboseSync {
		fmt.Printf("Verbose mode enabled\n")
		fmt.Printf("Platform: %s\n", osq.GetPlatform())
		fmt.Printf("Agent version: %s\n", cfg.Version)
	}

	// Initialize API client
	apiClient := api.NewClient(cfg, ds)

	// Set sync state to running
	if err := ds.SetSyncState(datastore.SyncStateRunning); err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}

	// Set last sync attempted timestamp
	if err := ds.SetLastSyncAttemptedAt(time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("failed to update last sync attempted: %w", err)
	}

	fmt.Println("Syncing system information with Drata...")

	// Get initialization data if needed
	if !ds.IsInitDataReady() {
		fmt.Println("Fetching initialization data...")
		if _, err := apiClient.GetInitData(); err != nil {
			if err := ds.SetSyncState(datastore.SyncStateError); err != nil {
				fmt.Printf("Warning: failed to update sync state: %v\n", err)
			}
			return fmt.Errorf("failed to get initialization data: %w", err)
		}
	}

	// Collect system information
	fmt.Println("Collecting system information...")
	queryResult, err := osq.GetSystemInfo(cfg.Version)
	if err != nil {
		if err := ds.SetSyncState(datastore.SyncStateError); err != nil {
			fmt.Printf("Warning: failed to update sync state: %v\n", err)
		}
		return fmt.Errorf("failed to collect system information: %w", err)
	}

	// Mark as manual run if forced
	queryResult.ManualRun = forceSync

	// Send to Drata
	fmt.Println("Sending data to Drata...")
	_, err = apiClient.Sync(queryResult)
	if err != nil {
		if err := ds.SetSyncState(datastore.SyncStateError); err != nil {
			fmt.Printf("Warning: failed to update sync state: %v\n", err)
		}
		return fmt.Errorf("failed to sync: %w", err)
	}

	// Update sync state
	if err := ds.SetSyncState(datastore.SyncStateSuccess); err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}

	fmt.Println("✓ Sync completed successfully!")

	// Show last checked time
	lastChecked := ds.GetLastCheckedAt()
	if lastChecked != "" {
		fmt.Printf("Last successful sync: %s\n", lastChecked)
	}

	return nil
}
