// Package osquery provides system query functionality using osquery.
package osquery

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Platform represents the operating system platform.
type Platform string

const (
	PlatformMacOS   Platform = "MACOS"
	PlatformWindows Platform = "WINDOWS"
	PlatformLinux   Platform = "LINUX"
)

// QueryResult represents the result of a system query.
type QueryResult struct {
	DrataAgentVersion string                 `json:"drataAgentVersion"`
	Platform          Platform               `json:"platform"`
	ManualRun         bool                   `json:"manualRun,omitempty"`
	RawQueryResults   map[string]interface{} `json:"rawQueryResults"`
}

// AgentDeviceIdentifiers represents the device identifiers used for registration.
type AgentDeviceIdentifiers struct {
	HWSerial struct {
		HardwareSerial string `json:"hardware_serial,omitempty"`
		BoardSerial    string `json:"board_serial,omitempty"`
	} `json:"hwSerial"`
	MacAddress struct {
		Mac string `json:"mac,omitempty"`
	} `json:"macAddress"`
}

// Client provides osquery functionality.
type Client struct {
	binaryPath string
	platform   Platform
	verbose    bool
}

// NewClient creates a new osquery client.
func NewClient(binaryPath string) (*Client, error) {
	return NewClientWithVerbose(binaryPath, false)
}

// NewClientWithVerbose creates a new osquery client with verbose option.
func NewClientWithVerbose(binaryPath string, verbose bool) (*Client, error) {
	platform, err := detectPlatform()
	if err != nil {
		return nil, err
	}

	// If no binary path specified, try to find osqueryi
	if binaryPath == "" {
		binaryPath, err = findOsqueryBinary()
		if err != nil {
			return nil, fmt.Errorf("osquery binary not found: %w", err)
		}
	}

	return &Client{
		binaryPath: binaryPath,
		platform:   platform,
		verbose:    verbose,
	}, nil
}

// SetVerbose sets the verbose mode for the client.
func (c *Client) SetVerbose(verbose bool) {
	c.verbose = verbose
}

// IsVerbose returns whether verbose mode is enabled.
func (c *Client) IsVerbose() bool {
	return c.verbose
}

// logVerbose prints a message if verbose mode is enabled.
func (c *Client) logVerbose(format string, args ...interface{}) {
	if c.verbose {
		fmt.Printf("[VERBOSE] "+format+"\n", args...)
	}
}

// detectPlatform detects the current platform.
func detectPlatform() (Platform, error) {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMacOS, nil
	case "windows":
		return PlatformWindows, nil
	case "linux":
		return PlatformLinux, nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// findOsqueryBinary attempts to find the osquery binary.
func findOsqueryBinary() (string, error) {
	binaryName := "osqueryi"
	if runtime.GOOS == "windows" {
		binaryName = "osqueryi.exe"
	}

	// Build list of paths to search
	var searchPaths []string

	// Add user home directory paths (important for Flatpak/immutable OS)
	if home := os.Getenv("HOME"); home != "" {
		searchPaths = append(searchPaths,
			filepath.Join(home, ".local", "bin", "osqueryi"),
			filepath.Join(home, ".local", "lib", "drata-agent", "bin", "osqueryi"),
		)
	}

	// Flatpak-specific paths
	searchPaths = append(searchPaths,
		"/app/bin/osqueryi",
		"/app/lib/drata-agent/bin/osqueryi",
	)

	// Common system locations
	commonPaths := []string{
		"/usr/local/bin/osqueryi",
		"/usr/bin/osqueryi",
		"/opt/osquery/bin/osqueryi",
		"/usr/lib/drata-agent/bin/osqueryi",
		"/usr/lib64/drata-agent/bin/osqueryi",
		"C:\\Program Files\\osquery\\osqueryi.exe",
		"C:\\ProgramData\\osquery\\osqueryi.exe",
	}
	searchPaths = append(searchPaths, commonPaths...)

	// First, try PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		return path, nil
	}

	// Then check all configured paths
	for _, path := range searchPaths {
		if fileExists(path) {
			return path, nil
		}
	}

	// Build error message with all searched paths
	return "", fmt.Errorf("%s not found in PATH or common locations. Searched paths:\n  - PATH lookup for '%s'\n  - %s",
		binaryName, binaryName, strings.Join(searchPaths, "\n  - "))
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetPlatform returns the detected platform.
func (c *Client) GetPlatform() Platform {
	return c.platform
}

// RunQuery executes an osquery SQL query and returns the JSON result.
func (c *Client) RunQuery(query string) ([]map[string]interface{}, error) {
	c.logVerbose("Executing osquery: %s", query)
	cmd := exec.Command(c.binaryPath, "--json", query)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			c.logVerbose("Query failed: %s", string(exitErr.Stderr))
			return nil, fmt.Errorf("osquery error: %s", string(exitErr.Stderr))
		}
		c.logVerbose("Query failed: %v", err)
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		c.logVerbose("Failed to parse output: %v", err)
		return nil, fmt.Errorf("failed to parse osquery output: %w", err)
	}

	c.logVerbose("Query returned %d results", len(result))
	return result, nil
}

// RunCommand executes a shell command and returns the output.
// SECURITY NOTE: This method should only be called with trusted, predefined commands
// from the platform-specific system query implementations (macos.go, linux.go, windows.go).
// Commands are hardcoded system utilities and should never include user input.
func (c *Client) RunCommand(command string) (string, error) {
	c.logVerbose("Executing command: %s", command)
	var cmd *exec.Cmd

	switch c.platform {
	case PlatformWindows:
		// Use UTF-8 code page for Windows
		fullCmd := fmt.Sprintf("cmd /c chcp 65001>nul && %s", command)
		cmd = exec.Command("cmd", "/c", fullCmd)
	default:
		cmd = exec.Command("sh", "-c", command)
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			c.logVerbose("Command failed: %s", string(exitErr.Stderr))
			return "", fmt.Errorf("command error: %s", string(exitErr.Stderr))
		}
		c.logVerbose("Command failed: %v", err)
		return "", err
	}

	result := strings.TrimSpace(string(output))
	c.logVerbose("Command output length: %d chars", len(result))
	return result, nil
}

// GetSystemInfo collects comprehensive system information.
func (c *Client) GetSystemInfo(version string) (*QueryResult, error) {
	switch c.platform {
	case PlatformMacOS:
		return c.getMacOSSystemInfo(version)
	case PlatformWindows:
		return c.getWindowsSystemInfo(version)
	case PlatformLinux:
		return c.getLinuxSystemInfo(version)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", c.platform)
	}
}

// GetAgentDeviceIdentifiers returns the device identifiers for registration.
func (c *Client) GetAgentDeviceIdentifiers() (*AgentDeviceIdentifiers, error) {
	switch c.platform {
	case PlatformMacOS:
		return c.getMacOSDeviceIdentifiers()
	case PlatformWindows:
		return c.getWindowsDeviceIdentifiers()
	case PlatformLinux:
		return c.getLinuxDeviceIdentifiers()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", c.platform)
	}
}

// GetDebugInfo returns debug information about the system.
func (c *Client) GetDebugInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// osquery version
	result, err := c.RunQuery("SELECT version FROM osquery_info")
	if err == nil && len(result) > 0 {
		info["osquery"] = result[0]
	}

	// OS version
	result, err = c.RunQuery("SELECT version, build, platform FROM os_version")
	if err == nil && len(result) > 0 {
		info["os"] = result[0]
	}

	// System info
	identifiers, err := c.GetAgentDeviceIdentifiers()
	if err == nil {
		info["system_info"] = identifiers
	}

	return info, nil
}

// Helper function to get the first result from a query
func (c *Client) queryFirst(query string) (map[string]interface{}, error) {
	result, err := c.RunQuery(query)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result[0], nil
	}
	return nil, nil
}

// Helper function to run multiple queries and flatten results
func (c *Client) queryAll(queries []string) ([]map[string]interface{}, error) {
	var allResults []map[string]interface{}

	for _, query := range queries {
		result, err := c.RunQuery(query)
		if err != nil {
			continue
		}
		allResults = append(allResults, result...)
	}

	return allResults, nil
}
