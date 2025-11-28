package cmd

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/drata/drata-agent-cli/internal/api"
	"github.com/drata/drata-agent-cli/internal/config"
	"github.com/drata/drata-agent-cli/internal/datastore"
	"github.com/drata/drata-agent-cli/internal/osquery"
)

var registerCmd = &cobra.Command{
	Use:   "register [token]",
	Short: "Register the agent with Drata",
	Long: `Register this device with Drata using a magic link token.

To get a registration token:
1. Log in to Drata at https://app.drata.com
2. Go to My Drata > Install the Drata Agent
3. Click "Register Drata Agent"
4. Copy the token from the magic link URL

Example:
  drata-agent register YOUR_TOKEN --region NA`,
	Args: cobra.ExactArgs(1),
	RunE: runRegister,
}

func init() {
	rootCmd.AddCommand(registerCmd)
	registerCmd.Flags().StringP("region", "r", "NA", "Drata region (NA, EU, APAC)")
}

func runRegister(cmd *cobra.Command, args []string) error {
	token := args[0]

	// Get region from flag
	regionStr, _ := cmd.Flags().GetString("region")
	region, err := config.ParseRegion(regionStr)
	if err != nil {
		return err
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg.Region = region

	// If environment flag is set, use it
	if targetEnv != "" {
		env, err := config.ParseTargetEnv(targetEnv)
		if err != nil {
			return err
		}
		cfg.TargetEnv = env
	}

	// Initialize data store
	ds, err := datastore.New()
	if err != nil {
		return fmt.Errorf("failed to initialize data store: %w", err)
	}

	// Check if already registered
	if ds.IsRegistered() {
		return fmt.Errorf("agent is already registered. Use 'drata-agent unregister' first")
	}

	// Set region and UUID
	if err := ds.SetRegion(region); err != nil {
		return fmt.Errorf("failed to set region: %w", err)
	}
	if ds.GetUUID() == "" {
		if err := ds.SetUUID(uuid.New().String()); err != nil {
			return fmt.Errorf("failed to set UUID: %w", err)
		}
	}

	// Initialize osquery client
	osq, err := osquery.NewClient(cfg.OsqueryPath)
	if err != nil {
		return fmt.Errorf("failed to initialize osquery: %w", err)
	}

	// Initialize API client
	apiClient := api.NewClient(cfg, ds)

	fmt.Printf("Registering agent with Drata (%s region)...\n", region)

	// Authenticate with magic link
	user, err := apiClient.LoginWithMagicLink(token)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("Authenticated as: %s %s (%s)\n", user.FirstName, user.LastName, user.Email)

	// Get device identifiers
	identifiers, err := osq.GetAgentDeviceIdentifiers()
	if err != nil {
		return fmt.Errorf("failed to get device identifiers: %w", err)
	}

	// Register device
	_, err = apiClient.Register(identifiers)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Set app version
	if err := ds.SetAppVersion(cfg.Version); err != nil {
		return fmt.Errorf("failed to set app version: %w", err)
	}

	fmt.Println("✓ Agent registered successfully!")
	fmt.Println()
	fmt.Println("You can now run 'drata-agent sync' to sync your system information.")
	fmt.Println("To run periodic syncs, use 'drata-agent daemon'.")

	return nil
}

func parseRegionFromToken(token string) config.Region {
	// Token might contain region information
	tokenLower := strings.ToLower(token)
	if strings.Contains(tokenLower, "eu.") || strings.Contains(tokenLower, "/eu/") {
		return config.RegionEU
	}
	if strings.Contains(tokenLower, "apac.") || strings.Contains(tokenLower, "/apac/") {
		return config.RegionAPAC
	}
	return config.RegionNA
}
