package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Region != RegionNA {
		t.Errorf("expected default region to be NA, got %s", cfg.Region)
	}

	if cfg.TargetEnv != EnvProd {
		t.Errorf("expected default env to be PROD, got %s", cfg.TargetEnv)
	}

	if cfg.SyncIntervalHours != 2 {
		t.Errorf("expected default sync interval to be 2, got %d", cfg.SyncIntervalHours)
	}
}

func TestAPIHostURL(t *testing.T) {
	tests := []struct {
		name     string
		env      TargetEnv
		region   Region
		expected string
	}{
		{"PROD NA", EnvProd, RegionNA, "https://agent.drata.com"},
		{"PROD EU", EnvProd, RegionEU, "https://agent.eu.drata.com"},
		{"PROD APAC", EnvProd, RegionAPAC, "https://agent.apac.drata.com"},
		{"DEV NA", EnvDev, RegionNA, "https://agent.dev.drata.com"},
		{"LOCAL NA", EnvLocal, RegionNA, "http://localhost:3000"},
		{"LOCAL EU", EnvLocal, RegionEU, "http://localhost:3001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				TargetEnv: tt.env,
				Region:    tt.region,
			}

			result := cfg.APIHostURL()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestWebAppURL(t *testing.T) {
	tests := []struct {
		name     string
		env      TargetEnv
		expected string
	}{
		{"PROD", EnvProd, "https://app.drata.com"},
		{"DEV", EnvDev, "https://app.dev.drata.com"},
		{"QA", EnvQA, "https://app.qa.drata.com"},
		{"LOCAL", EnvLocal, "http://localhost:5000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				TargetEnv: tt.env,
			}

			result := cfg.WebAppURL()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseRegion(t *testing.T) {
	tests := []struct {
		input    string
		expected Region
		hasError bool
	}{
		{"NA", RegionNA, false},
		{"EU", RegionEU, false},
		{"APAC", RegionAPAC, false},
		{"na", RegionNA, false},
		{"eu", RegionEU, false},
		{"apac", RegionAPAC, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseRegion(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input %s", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestParseTargetEnv(t *testing.T) {
	tests := []struct {
		input    string
		expected TargetEnv
		hasError bool
	}{
		{"LOCAL", EnvLocal, false},
		{"DEV", EnvDev, false},
		{"QA", EnvQA, false},
		{"PROD", EnvProd, false},
		{"local", EnvLocal, false},
		{"prod", EnvProd, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseTargetEnv(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input %s", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestGetDataDir(t *testing.T) {
	dir, err := GetDataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	expected := filepath.Join(homeDir, ".drata-agent", "data")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}

	// Verify directory was created
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("data directory was not created")
	}
}
