// Package cmd provides the CLI commands for the Drata Agent.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Used for flags
	cfgFile   string
	region    string
	targetEnv string

	rootCmd = &cobra.Command{
		Use:   "drata-agent",
		Short: "Drata Agent CLI - Compliance monitoring agent",
		Long: `The Drata Agent CLI is a command-line version of the Drata Agent 
that monitors your system's security configuration for SOC 2 compliance.

This agent collects read-only information about your system's security settings
including screensaver locking, password manager, antivirus software, and 
automatic updates.

For more information, visit https://help.drata.com/`,
		Version: "3.9.0-cli",
	}
)

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.drata-agent/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&region, "region", "", "Drata region (NA, EU, APAC)")
	rootCmd.PersistentFlags().StringVar(&targetEnv, "env", "", "Target environment (LOCAL, DEV, QA, PROD)")
}
