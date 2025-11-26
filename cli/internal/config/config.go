// Package config provides configuration management for the Drata Agent CLI.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Region represents the Drata API region.
type Region string

const (
	RegionNA   Region = "NA"
	RegionEU   Region = "EU"
	RegionAPAC Region = "APAC"
)

// TargetEnv represents the target environment.
type TargetEnv string

const (
	EnvLocal TargetEnv = "LOCAL"
	EnvDev   TargetEnv = "DEV"
	EnvQA    TargetEnv = "QA"
	EnvProd  TargetEnv = "PROD"
)

// Config holds all configuration for the CLI.
type Config struct {
	// API configuration
	Region    Region    `mapstructure:"region"`
	TargetEnv TargetEnv `mapstructure:"target_env"`

	// Sync configuration
	SyncIntervalHours      int `mapstructure:"sync_interval_hours"`
	MinHoursSinceLastSync  int `mapstructure:"min_hours_since_last_sync"`
	MinMinutesBetweenSyncs int `mapstructure:"min_minutes_between_syncs"`

	// osquery configuration
	OsqueryPath string `mapstructure:"osquery_path"`

	// CLI version
	Version string `mapstructure:"version"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Region:                 RegionNA,
		TargetEnv:              EnvProd,
		SyncIntervalHours:      2,
		MinHoursSinceLastSync:  24,
		MinMinutesBetweenSyncs: 15,
		OsqueryPath:            "",
		Version:                "3.9.0-cli",
	}
}

// APIHostURL returns the API host URL based on environment and region.
func (c *Config) APIHostURL() string {
	apiURLs := map[TargetEnv]map[Region]string{
		EnvLocal: {
			RegionNA:   "http://localhost:3000",
			RegionEU:   "http://localhost:3001",
			RegionAPAC: "http://localhost:3002",
		},
		EnvDev: {
			RegionNA:   "https://agent.dev.drata.com",
			RegionEU:   "https://agent.dev.drata.com",
			RegionAPAC: "https://agent.dev.drata.com",
		},
		EnvQA: {
			RegionNA:   "https://agent.qa.drata.com",
			RegionEU:   "https://agent.qa.drata.com",
			RegionAPAC: "https://agent.qa.drata.com",
		},
		EnvProd: {
			RegionNA:   "https://agent.drata.com",
			RegionEU:   "https://agent.eu.drata.com",
			RegionAPAC: "https://agent.apac.drata.com",
		},
	}

	if envURLs, ok := apiURLs[c.TargetEnv]; ok {
		if url, ok := envURLs[c.Region]; ok {
			return url
		}
	}

	return apiURLs[EnvProd][RegionNA]
}

// WebAppURL returns the web application URL based on environment.
func (c *Config) WebAppURL() string {
	webAppURLs := map[TargetEnv]string{
		EnvLocal: "http://localhost:5000",
		EnvDev:   "https://app.dev.drata.com",
		EnvQA:    "https://app.qa.drata.com",
		EnvProd:  "https://app.drata.com",
	}

	if url, ok := webAppURLs[c.TargetEnv]; ok {
		return url
	}

	return webAppURLs[EnvProd]
}

// Load loads configuration from file and environment variables.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Set up viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config paths
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	viper.AddConfigPath(configDir)
	viper.AddConfigPath(".")

	// Environment variable support
	viper.SetEnvPrefix("DRATA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	// Unmarshal into config struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// Save saves the configuration to file.
func (c *Config) Save() error {
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set values in viper
	viper.Set("region", string(c.Region))
	viper.Set("target_env", string(c.TargetEnv))
	viper.Set("sync_interval_hours", c.SyncIntervalHours)
	viper.Set("min_hours_since_last_sync", c.MinHoursSinceLastSync)
	viper.Set("min_minutes_between_syncs", c.MinMinutesBetweenSyncs)
	viper.Set("osquery_path", c.OsqueryPath)
	viper.Set("version", c.Version)

	configPath := filepath.Join(configDir, "config.yaml")
	return viper.WriteConfigAs(configPath)
}

// getConfigDir returns the configuration directory path.
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".drata-agent"), nil
}

// GetDataDir returns the data directory path.
func GetDataDir() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	dataDir := filepath.Join(configDir, "data")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return "", err
	}
	return dataDir, nil
}

// ParseRegion parses a string into a Region.
func ParseRegion(s string) (Region, error) {
	switch strings.ToUpper(s) {
	case "NA":
		return RegionNA, nil
	case "EU":
		return RegionEU, nil
	case "APAC":
		return RegionAPAC, nil
	default:
		return "", fmt.Errorf("invalid region: %s (valid: NA, EU, APAC)", s)
	}
}

// ParseTargetEnv parses a string into a TargetEnv.
func ParseTargetEnv(s string) (TargetEnv, error) {
	switch strings.ToUpper(s) {
	case "LOCAL":
		return EnvLocal, nil
	case "DEV":
		return EnvDev, nil
	case "QA":
		return EnvQA, nil
	case "PROD":
		return EnvProd, nil
	default:
		return "", fmt.Errorf("invalid environment: %s (valid: LOCAL, DEV, QA, PROD)", s)
	}
}
