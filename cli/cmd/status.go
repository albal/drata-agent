package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/drata/drata-agent-cli/internal/config"
	"github.com/drata/drata-agent-cli/internal/datastore"
	"github.com/drata/drata-agent-cli/internal/osquery"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show agent status",
	Long: `Display the current status of the Drata Agent.

This shows:
- Registration status
- User information (if registered)
- Last sync time and status
- System information

Example:
  drata-agent status`,
	RunE: runStatus,
}

var verboseStatus bool

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVarP(&verboseStatus, "verbose", "v", false, "Show detailed system information")
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	fmt.Println("Drata Agent CLI Status")
	fmt.Println("======================")
	fmt.Println()

	// Version info
	fmt.Printf("Version: %s\n", cfg.Version)
	fmt.Printf("Environment: %s\n", cfg.TargetEnv)
	fmt.Printf("Region: %s\n", ds.GetRegion())
	fmt.Printf("API Endpoint: %s\n", cfg.APIHostURL())
	fmt.Println()

	// Registration status
	fmt.Println("Registration")
	fmt.Println("------------")
	if ds.IsRegistered() {
		fmt.Println("Status: ✓ Registered")
		user := ds.GetUser()
		if user != nil {
			fmt.Printf("User: %s %s\n", user.FirstName, user.LastName)
			fmt.Printf("Email: %s\n", user.Email)
			if user.JobTitle != "" {
				fmt.Printf("Job Title: %s\n", user.JobTitle)
			}
		}
	} else {
		fmt.Println("Status: ✗ Not registered")
		fmt.Println()
		fmt.Println("To register, run:")
		fmt.Printf("  drata-agent register YOUR_TOKEN --region %s\n", cfg.Region)
	}
	fmt.Println()

	// Sync status
	fmt.Println("Sync Status")
	fmt.Println("-----------")
	syncState := ds.GetSyncState()
	switch syncState {
	case datastore.SyncStateSuccess:
		fmt.Println("Last Sync: ✓ Success")
	case datastore.SyncStateError:
		fmt.Println("Last Sync: ✗ Error")
	case datastore.SyncStateRunning:
		fmt.Println("Last Sync: ⋯ In Progress")
	case datastore.SyncStateUnknown:
		fmt.Println("Last Sync: ? Unknown")
	default:
		fmt.Println("Last Sync: Never synced")
	}

	lastChecked := ds.GetLastCheckedAt()
	if lastChecked != "" {
		if t, err := time.Parse(time.RFC3339, lastChecked); err == nil {
			fmt.Printf("Last Success: %s (%s ago)\n", t.Local().Format(time.RFC1123), formatDuration(time.Since(t)))
		} else {
			fmt.Printf("Last Success: %s\n", lastChecked)
		}
	}

	lastAttempt := ds.GetLastSyncAttemptedAt()
	if lastAttempt != "" {
		if t, err := time.Parse(time.RFC3339, lastAttempt); err == nil {
			fmt.Printf("Last Attempt: %s (%s ago)\n", t.Local().Format(time.RFC1123), formatDuration(time.Since(t)))
		} else {
			fmt.Printf("Last Attempt: %s\n", lastAttempt)
		}
	}
	fmt.Println()

	// System information
	if verboseStatus {
		fmt.Println("System Information")
		fmt.Println("------------------")

		osq, err := osquery.NewClient(cfg.OsqueryPath)
		if err != nil {
			fmt.Printf("Warning: Could not initialize osquery: %v\n", err)
		} else {
			fmt.Printf("Platform: %s\n", osq.GetPlatform())

			debugInfo, err := osq.GetDebugInfo()
			if err != nil {
				fmt.Printf("Warning: Could not get debug info: %v\n", err)
			} else {
				if osqInfo, ok := debugInfo["osquery"].(map[string]interface{}); ok {
					if version, ok := osqInfo["version"].(string); ok {
						fmt.Printf("osquery Version: %s\n", version)
					}
				}
				if osInfo, ok := debugInfo["os"].(map[string]interface{}); ok {
					if platform, ok := osInfo["platform"].(string); ok {
						fmt.Printf("OS Platform: %s\n", platform)
					}
					if version, ok := osInfo["version"].(string); ok {
						fmt.Printf("OS Version: %s\n", version)
					}
				}
			}

			identifiers, err := osq.GetAgentDeviceIdentifiers()
			if err != nil {
				fmt.Printf("Warning: Could not get device identifiers: %v\n", err)
			} else {
				if identifiers.HWSerial.HardwareSerial != "" {
					fmt.Printf("Hardware Serial: %s\n", identifiers.HWSerial.HardwareSerial)
				}
				if identifiers.MacAddress.Mac != "" {
					fmt.Printf("MAC Address: %s\n", identifiers.MacAddress.Mac)
				}
			}
		}
		fmt.Println()
	}

	// Configuration
	fmt.Println("Configuration")
	fmt.Println("-------------")
	fmt.Printf("Sync Interval: %d hours\n", cfg.SyncIntervalHours)
	fmt.Printf("Min Hours Since Last Sync: %d\n", cfg.MinHoursSinceLastSync)
	fmt.Printf("Min Minutes Between Syncs: %d\n", cfg.MinMinutesBetweenSyncs)
	if cfg.OsqueryPath != "" {
		fmt.Printf("osquery Path: %s\n", cfg.OsqueryPath)
	} else {
		fmt.Println("osquery Path: (auto-detect)")
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
