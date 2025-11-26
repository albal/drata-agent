package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/drata/drata-agent-cli/internal/datastore"
)

var unregisterCmd = &cobra.Command{
	Use:   "unregister",
	Short: "Unregister the agent",
	Long: `Unregister this device from Drata.

This clears all local registration data and credentials.
The device will need to be registered again to sync with Drata.

Example:
  drata-agent unregister`,
	RunE: runUnregister,
}

var confirmUnregister bool

func init() {
	rootCmd.AddCommand(unregisterCmd)
	unregisterCmd.Flags().BoolVarP(&confirmUnregister, "yes", "y", false, "Skip confirmation prompt")
}

func runUnregister(cmd *cobra.Command, args []string) error {
	// Initialize data store
	ds, err := datastore.New()
	if err != nil {
		return fmt.Errorf("failed to initialize data store: %w", err)
	}

	// Check if registered
	if !ds.IsRegistered() {
		fmt.Println("Agent is not currently registered.")
		return nil
	}

	// Confirm unregistration
	if !confirmUnregister {
		user := ds.GetUser()
		if user != nil {
			fmt.Printf("Currently registered as: %s (%s)\n", user.Email, user.FirstName+" "+user.LastName)
		}
		fmt.Print("Are you sure you want to unregister? [y/N]: ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil || (response != "y" && response != "Y" && response != "yes" && response != "Yes") {
			fmt.Println("Unregistration cancelled.")
			return nil
		}
	}

	// Clear all data
	if err := ds.Clear(); err != nil {
		return fmt.Errorf("failed to clear data: %w", err)
	}

	fmt.Println("✓ Agent unregistered successfully.")
	fmt.Println()
	fmt.Println("To register again, run:")
	fmt.Println("  drata-agent register YOUR_TOKEN --region NA")

	return nil
}
