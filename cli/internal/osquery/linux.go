package osquery

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

func isValidSessionUser(user string) bool {
	if user == "" || user == "root" {
		return false
	}
	for _, r := range user {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func (c *Client) getDesktopSessionUser() string {
	candidates := []string{
		os.Getenv("SUDO_USER"),
		os.Getenv("LOGNAME"),
		os.Getenv("USER"),
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if isValidSessionUser(candidate) {
			return candidate
		}
	}
	if output, err := c.RunCommand("logname"); err == nil {
		candidate := strings.TrimSpace(output)
		if isValidSessionUser(candidate) {
			return candidate
		}
	}
	return ""
}

func (c *Client) runGsettingsCommand(args string) (string, error) {
	baseCmd := fmt.Sprintf("gsettings %s", args)
	user := c.getDesktopSessionUser()
	if user == "" {
		return c.RunCommand(baseCmd)
	}

	userCmd := fmt.Sprintf("uid=$(id -u %[1]s) && sudo -u %[1]s env XDG_RUNTIME_DIR=/run/user/$uid DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus %s", user, baseCmd)
	if output, err := c.RunCommand(userCmd); err == nil {
		return output, nil
	}

	return c.RunCommand(baseCmd)
}

func parseGsettingsUint(output string) (int, error) {
	value := strings.TrimSpace(output)
	if value == "" {
		return 0, fmt.Errorf("empty gsettings output")
	}
	// gsettings often returns "uint32 300"; take the last token as numeric value
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid gsettings output: %s", output)
	}
	return strconv.Atoi(parts[len(parts)-1])
}

// isRPMBasedDistro checks if the system is RPM-based (Fedora/RHEL/CentOS).
func (c *Client) isRPMBasedDistro() bool {
	// Check if /etc/redhat-release or /etc/fedora-release exists
	if _, err := os.Stat("/etc/redhat-release"); err == nil {
		return true
	}
	if _, err := os.Stat("/etc/fedora-release"); err == nil {
		return true
	}
	// Check for dnf or yum package managers
	if _, err := os.Stat("/usr/bin/dnf"); err == nil {
		return true
	}
	if _, err := os.Stat("/usr/bin/yum"); err == nil {
		return true
	}
	return false
}

// getLinuxSystemInfo collects Linux-specific system information.
func (c *Client) getLinuxSystemInfo(version string) (*QueryResult, error) {
	rawResults := make(map[string]interface{})

	// OS Version
	if result, err := c.queryFirst("SELECT name, version, platform FROM os_version"); err == nil && result != nil {
		rawResults["osVersion"] = result
	}

	// Hardware Serial
	if result, err := c.queryFirst("SELECT hardware_serial FROM system_info"); err == nil && result != nil {
		rawResults["hwSerial"] = result
	}

	// Hardware Model
	if result, err := c.queryFirst("SELECT hardware_model FROM system_info"); err == nil && result != nil {
		rawResults["hwModel"] = result
	}

	// System Information
	if result, err := c.queryFirst("SELECT board_serial, board_model, computer_name, hostname, local_hostname FROM system_info"); err == nil && result != nil {
		rawResults["boardSerial"] = result["board_serial"]
		rawResults["boardModel"] = result["board_model"]
		rawResults["computerName"] = result["computer_name"]
		rawResults["hostName"] = result["hostname"]
		rawResults["localHostName"] = result["local_hostname"]
	}

	// Firewall Status - try both firewalld (RHEL/Fedora) and UFW (Debian/Ubuntu)
	if c.isRPMBasedDistro() {
		// Firewalld for RHEL/Fedora
		if output, err := c.RunCommand("systemctl is-active firewalld"); err == nil {
			rawResults["firewallStatus"] = map[string]interface{}{
				"passed": output == "active",
				"type":   "firewalld",
				"status": output,
			}
		}
	} else {
		// UFW for Debian/Ubuntu
		if result, err := c.queryFirst("SELECT COUNT(*) AS passed FROM augeas WHERE path = '/etc/ufw/ufw.conf' AND label = 'ENABLED' AND value = 'yes'"); err == nil && result != nil {
			rawResults["firewallStatus"] = result
		}
	}

	// Application List - try both rpm_packages and deb_packages
	if c.isRPMBasedDistro() {
		if result, err := c.RunQuery("SELECT name, version FROM rpm_packages"); err == nil {
			rawResults["appList"] = result
		}
	} else {
		if result, err := c.RunQuery("SELECT name, version FROM deb_packages"); err == nil {
			rawResults["appList"] = result
		}
	}

	// Antivirus Check - check for clamav and flatpak-installed clam apps
	antivirusStatus := map[string]interface{}{"passed": false}
	if c.isRPMBasedDistro() {
		// Check for clamav daemon
		if output, err := c.RunCommand("rpm -q clamav"); err == nil && output != "" {
			antivirusStatus["clamav"] = map[string]interface{}{
				"installed": true,
				"version":   output,
			}
			antivirusStatus["passed"] = true
		}
	} else {
		// Check for clamav on Debian/Ubuntu
		if output, err := c.RunCommand("dpkg -l clamav | grep -E '^ii'"); err == nil && output != "" {
			antivirusStatus["clamav"] = map[string]interface{}{
				"installed": true,
				"version":   output,
			}
			antivirusStatus["passed"] = true
		}
	}
	// Check for clamtk/clamav installed via Flatpak (system or user scope)
	flatpakScopes := []struct {
		label   string
		command string
	}{
		{"system", "flatpak list --app --system | grep -i clam"},
		{"user", "flatpak list --app --user | grep -i clam"},
	}
	for _, scope := range flatpakScopes {
		if output, err := c.RunCommand(scope.command); err == nil && output != "" {
			antivirusStatus["flatpak"] = map[string]interface{}{
				"installed": true,
				"scope":     scope.label,
				"details":   output,
			}
			antivirusStatus["passed"] = true
			break
		}
	}
	rawResults["antivirusStatus"] = antivirusStatus

	// Browser Extensions - use user home directory paths
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/root"
	}
	extensions := make([]interface{}, 0)

	// Firefox addons - check user profile directory
	firefoxPath := filepath.Join(homeDir, ".mozilla", "firefox")
	if _, err := os.Stat(firefoxPath); err == nil {
		if result, err := c.RunQuery("SELECT name FROM firefox_addons"); err == nil {
			for _, r := range result {
				extensions = append(extensions, r)
			}
		}
	}

	// Chrome extensions - check user profile directory
	chromePath := filepath.Join(homeDir, ".config", "google-chrome")
	if _, err := os.Stat(chromePath); err == nil {
		if result, err := c.RunQuery("SELECT name FROM chrome_extensions"); err == nil {
			for _, r := range result {
				extensions = append(extensions, r)
			}
		}
	}
	rawResults["browserExtensions"] = extensions

	// MAC Address
	if result, err := c.queryFirst("SELECT mac FROM interface_details WHERE interface in (SELECT DISTINCT interface FROM interface_addresses WHERE interface NOT IN ('lo')) LIMIT 1"); err == nil && result != nil {
		rawResults["macAddress"] = result
	}

	// Auto Update Settings - use only gsettings for autoUpdateEnabled check
	autoUpdateSettings := make([]interface{}, 0)
	// GNOME Software automatic updates check - only this sets autoUpdateEnabled
	if output, err := c.runGsettingsCommand("get org.gnome.software download-updates"); err == nil {
		autoUpdateSettings = append(autoUpdateSettings, map[string]string{"gnomeSoftwareDownloadUpdates": output})
		if output == "true" {
			rawResults["autoUpdateEnabled"] = map[string]interface{}{"passed": 1}
		}
	}
	// Collect additional distro-specific update settings for informational purposes
	if c.isRPMBasedDistro() {
		if output, err := c.RunCommand("systemctl is-enabled dnf-automatic.timer || systemctl is-enabled yum-cron"); err == nil {
			autoUpdateSettings = append(autoUpdateSettings, map[string]string{"autoUpdateService": output})
		}
		if output, err := c.RunCommand("dnf history list --last 10 || yum history list last 10"); err == nil {
			autoUpdateSettings = append(autoUpdateSettings, map[string]string{"recentUpdates": output})
		}
		if output, err := c.RunCommand("cat /etc/dnf/automatic.conf || cat /etc/yum/yum-cron.conf"); err == nil {
			autoUpdateSettings = append(autoUpdateSettings, map[string]string{"autoUpdateConfig": output})
		}
		if output, err := c.RunCommand("rpm -q --last | head -20"); err == nil {
			autoUpdateSettings = append(autoUpdateSettings, map[string]string{"recentPackages": output})
		}
	} else {
		if output, err := c.RunCommand("apt-config dump | grep -E '^(APT::Periodic|Unattended-Upgrade)::'"); err == nil {
			autoUpdateSettings = append(autoUpdateSettings, map[string]string{"aptConfig": output})
		}
		if output, err := c.RunCommand("systemctl show apt-daily* --property=NextElapseUSecMonotonic,NextElapseUSecRealtime,Unit,Description,UnitFileState,LastTriggerUSec"); err == nil {
			autoUpdateSettings = append(autoUpdateSettings, map[string]string{"aptDailyStatus": output})
		}
		if output, err := c.RunCommand("journalctl --user -u apt-daily.service -u apt-daily-upgrade.service --since -7day -n 10 --no-pager --quiet || journalctl -u apt-daily.service -u apt-daily-upgrade.service --since -7day -n 10 --no-pager --quiet"); err == nil {
			autoUpdateSettings = append(autoUpdateSettings, map[string]string{"aptDailyLogs": output})
		}
	}
	rawResults["autoUpdateSettings"] = autoUpdateSettings

	// Screen Lock Status - only check idle-delay for Fedora/RHEL Gnome
	screenLockStatus := make([]interface{}, 0)
	// Capture idle-delay from org.gnome.desktop.session so we know when the screen saver triggers
	if output, err := c.runGsettingsCommand("get org.gnome.desktop.session idle-delay"); err == nil {
		if seconds, parseErr := parseGsettingsUint(output); parseErr == nil {
			screenLockStatus = append(screenLockStatus, map[string]interface{}{"idleDelaySeconds": seconds})
		} else {
			screenLockStatus = append(screenLockStatus, map[string]string{"idleDelay": output})
		}
	}
	// Check lock-delay from org.gnome.desktop.screensaver to ensure lock engages quickly
	if output, err := c.runGsettingsCommand("get org.gnome.desktop.screensaver lock-delay"); err == nil {
		if seconds, parseErr := parseGsettingsUint(output); parseErr == nil {
			screenLockStatus = append(screenLockStatus, map[string]interface{}{"lockDelaySeconds": seconds})
		} else {
			screenLockStatus = append(screenLockStatus, map[string]string{"lockDelay": output})
		}
	}
	rawResults["screenLockStatus"] = screenLockStatus

	// Location Services
	locationServices := make(map[string]interface{})
	if output, err := c.runGsettingsCommand("get org.gnome.system.location enabled"); err == nil {
		locationServices["gnomeLocation"] = output
	}
	rawResults["locationServices"] = locationServices

	// Screen Lock Settings - use gsettings which works for current user
	screenLockSettings := make(map[string]interface{})
	if output, err := c.runGsettingsCommand("list-recursively org.gnome.settings-daemon.plugins.power"); err == nil && output != "" {
		screenLockSettings["powerSettings"] = output
	}
	if output, err := c.runGsettingsCommand("list-recursively org.gnome.desktop.screensaver"); err == nil && output != "" {
		screenLockSettings["screenSettings"] = output
	}
	if output, err := c.runGsettingsCommand("list-recursively org.gnome.desktop.session"); err == nil && output != "" {
		screenLockSettings["sessionSettings"] = output
	}
	rawResults["screenLockSettings"] = screenLockSettings

	return &QueryResult{
		DrataAgentVersion: version,
		Platform:          PlatformLinux,
		RawQueryResults:   rawResults,
	}, nil
}

// getLinuxDeviceIdentifiers returns Linux device identifiers.
func (c *Client) getLinuxDeviceIdentifiers() (*AgentDeviceIdentifiers, error) {
	identifiers := &AgentDeviceIdentifiers{}

	if result, err := c.queryFirst("SELECT hardware_serial, board_serial FROM system_info"); err == nil && result != nil {
		if v, ok := result["hardware_serial"].(string); ok {
			identifiers.HWSerial.HardwareSerial = v
		}
		if v, ok := result["board_serial"].(string); ok {
			identifiers.HWSerial.BoardSerial = v
		}
	}

	if result, err := c.queryFirst("SELECT mac FROM interface_details WHERE interface in (SELECT DISTINCT interface FROM interface_addresses WHERE interface NOT IN ('lo')) LIMIT 1"); err == nil && result != nil {
		if v, ok := result["mac"].(string); ok {
			identifiers.MacAddress.Mac = v
		}
	}

	return identifiers, nil
}
