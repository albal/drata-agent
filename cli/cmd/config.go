package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/drata/drata-agent-cli/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage agent configuration",
	Long: `View and modify the Drata Agent configuration.

Configuration file location: $HOME/.drata-agent/config.yaml

Available configuration options:
- region: Drata region (NA, EU, APAC)
- target_env: Target environment (LOCAL, DEV, QA, PROD)
- sync_interval_hours: Sync interval in hours
- min_hours_since_last_sync: Minimum hours between syncs
- min_minutes_between_syncs: Minimum minutes between sync attempts
- osquery_path: Path to osquery binary (empty for auto-detect)

Example:
  drata-agent config show
  drata-agent config set region EU
  drata-agent config set sync_interval_hours 4`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE:  runConfigPath,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file with defaults",
	RunE:  runConfigInit,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Current Configuration")
	fmt.Println("=====================")
	fmt.Printf("region: %s\n", cfg.Region)
	fmt.Printf("target_env: %s\n", cfg.TargetEnv)
	fmt.Printf("sync_interval_hours: %d\n", cfg.SyncIntervalHours)
	fmt.Printf("min_hours_since_last_sync: %d\n", cfg.MinHoursSinceLastSync)
	fmt.Printf("min_minutes_between_syncs: %d\n", cfg.MinMinutesBetweenSyncs)
	if cfg.OsqueryPath != "" {
		fmt.Printf("osquery_path: %s\n", cfg.OsqueryPath)
	} else {
		fmt.Println("osquery_path: (auto-detect)")
	}
	fmt.Printf("version: %s\n", cfg.Version)

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch key {
	case "region":
		region, err := config.ParseRegion(value)
		if err != nil {
			return err
		}
		cfg.Region = region
	case "target_env":
		env, err := config.ParseTargetEnv(value)
		if err != nil {
			return err
		}
		cfg.TargetEnv = env
	case "sync_interval_hours":
		var interval int
		if _, err := fmt.Sscanf(value, "%d", &interval); err != nil || interval < 1 {
			return fmt.Errorf("sync_interval_hours must be a positive integer")
		}
		cfg.SyncIntervalHours = interval
	case "min_hours_since_last_sync":
		var hours int
		if _, err := fmt.Sscanf(value, "%d", &hours); err != nil || hours < 0 {
			return fmt.Errorf("min_hours_since_last_sync must be a non-negative integer")
		}
		cfg.MinHoursSinceLastSync = hours
	case "min_minutes_between_syncs":
		var minutes int
		if _, err := fmt.Sscanf(value, "%d", &minutes); err != nil || minutes < 0 {
			return fmt.Errorf("min_minutes_between_syncs must be a non-negative integer")
		}
		cfg.MinMinutesBetweenSyncs = minutes
	case "osquery_path":
		cfg.OsqueryPath = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Set %s = %s\n", key, value)
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".drata-agent", "config.yaml")
	fmt.Println(configPath)
	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	cfg := config.DefaultConfig()

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".drata-agent", "config.yaml")
	fmt.Printf("✓ Configuration initialized at: %s\n", configPath)
	return nil
}
