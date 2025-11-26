package osquery

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

	// Firewall Status (UFW)
	if result, err := c.queryFirst("SELECT COUNT(*) AS passed FROM augeas WHERE path = '/etc/ufw/ufw.conf' AND label = 'ENABLED' AND value = 'yes'"); err == nil && result != nil {
		rawResults["firewallStatus"] = result
	}

	// Application List (deb packages)
	if result, err := c.RunQuery("SELECT name, version FROM deb_packages"); err == nil {
		rawResults["appList"] = result
	}

	// Browser Extensions
	extensions, _ := c.queryAll([]string{
		"SELECT name FROM firefox_addons",
		"SELECT name FROM chrome_extensions",
	})
	rawResults["browserExtensions"] = extensions

	// MAC Address
	if result, err := c.queryFirst("SELECT mac FROM interface_details WHERE interface in (SELECT DISTINCT interface FROM interface_addresses WHERE interface NOT IN ('lo')) LIMIT 1"); err == nil && result != nil {
		rawResults["macAddress"] = result
	}

	// Auto Update
	if result, err := c.queryFirst("SELECT COUNT(*) AS passed FROM file WHERE path = '/etc/apt/apt.conf.d/50unattended-upgrades'"); err == nil && result != nil {
		rawResults["autoUpdateEnabled"] = result
	}

	// Auto Update Settings
	autoUpdateSettings := make([]interface{}, 0)
	if output, err := c.RunCommand("apt-config dump | grep -E '^(APT::Periodic|Unattended-Upgrade)::'"); err == nil {
		autoUpdateSettings = append(autoUpdateSettings, output)
	}
	if output, err := c.RunCommand("systemctl show apt-daily* --property=NextElapseUSecMonotonic,NextElapseUSecRealtime,Unit,Description,UnitFileState,LastTriggerUSec"); err == nil {
		autoUpdateSettings = append(autoUpdateSettings, output)
	}
	if output, err := c.RunCommand("journalctl -u apt-daily.service -u apt-daily-upgrade.service --since -7day -n 10 --no-pager --quiet"); err == nil {
		autoUpdateSettings = append(autoUpdateSettings, output)
	}
	if output, err := c.RunCommand("/usr/lib/update-notifier/apt-check"); err == nil {
		autoUpdateSettings = append(autoUpdateSettings, output)
	}
	if output, err := c.RunCommand("awk '/^Start-Date:/ {block=\"\"; inblock=1} inblock {block = block $0 ORS} /^End-Date:/ {if (block ~ /Upgrade:/) last=block; inblock=0} END {print last}' /var/log/apt/history.log"); err == nil {
		autoUpdateSettings = append(autoUpdateSettings, output)
	}
	rawResults["autoUpdateSettings"] = autoUpdateSettings

	// Screen Lock Status
	screenLockStatus := make([]interface{}, 0)
	if output, err := c.RunCommand("gsettings get org.gnome.desktop.screensaver lock-delay"); err == nil {
		screenLockStatus = append(screenLockStatus, output)
	}
	if output, err := c.RunCommand("gsettings get org.gnome.desktop.screensaver lock-enabled"); err == nil {
		screenLockStatus = append(screenLockStatus, output)
	}
	rawResults["screenLockStatus"] = screenLockStatus

	// Location Services
	if output, err := c.RunCommand("gsettings get org.gnome.system.location enabled"); err == nil {
		rawResults["locationServices"] = map[string]interface{}{
			"commandResults": output,
		}
	}

	// Screen Lock Settings
	screenLockSettings := make(map[string]interface{})
	if output, err := c.RunCommand("gsettings list-recursively org.gnome.settings-daemon.plugins.power"); err == nil {
		screenLockSettings["powerSettings"] = output
	}
	if output, err := c.RunCommand("gsettings list-recursively org.gnome.desktop.screensaver"); err == nil {
		screenLockSettings["screenSettings"] = output
	}
	if output, err := c.RunCommand("gsettings list-recursively org.gnome.desktop.session"); err == nil {
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
