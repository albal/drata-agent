package osquery

// getMacOSSystemInfo collects macOS-specific system information.
func (c *Client) getMacOSSystemInfo(version string) (*QueryResult, error) {
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

	// HDD Encryption Status
	if result, err := c.queryFirst("SELECT de.encrypted FROM mounts m JOIN disk_encryption de on de.name=m.device WHERE m.path ='/'"); err == nil && result != nil {
		rawResults["hddEncryptionStatus"] = result
	}

	// FileVault Status
	if output, err := c.RunCommand("fdesetup status"); err == nil {
		rawResults["fileVaultEnabled"] = map[string]interface{}{
			"commandResults": output,
		}
	}

	// Firewall Status
	if result, err := c.queryFirst("SELECT global_state FROM alf"); err == nil && result != nil {
		rawResults["firewallStatus"] = result
	}

	// Application List
	if result, err := c.RunQuery("SELECT name, bundle_short_version, info_string FROM apps"); err == nil {
		rawResults["appList"] = result
	}

	// Browser Extensions
	extensions, _ := c.queryAll([]string{
		"SELECT name FROM firefox_addons",
		"SELECT name FROM chrome_extensions",
		"SELECT name FROM safari_extensions",
	})
	rawResults["browserExtensions"] = extensions

	// MAC Address
	if result, err := c.queryFirst("SELECT mac FROM interface_details WHERE interface in (SELECT DISTINCT interface FROM interface_addresses WHERE interface IN ('en0', 'en1')) LIMIT 1"); err == nil && result != nil {
		rawResults["macAddress"] = result
	}

	// Auto Update
	if output, err := c.RunCommand("softwareupdate --schedule"); err == nil {
		value := "0"
		if containsIgnoreCase(output, "turned on") {
			value = "1"
		}
		rawResults["autoUpdateEnabled"] = map[string]interface{}{
			"value": value,
		}
	}

	// Gatekeeper
	if result, err := c.queryFirst("SELECT assessments_enabled FROM gatekeeper"); err == nil && result != nil {
		rawResults["gateKeeperEnabled"] = result
	}

	// Protection Settings
	protectionSettings := make(map[string]interface{})
	if result, err := c.RunQuery("SELECT assessments_enabled, dev_id_enabled FROM gatekeeper"); err == nil {
		protectionSettings["gatekeeper"] = result
	}
	if output, err := c.RunCommand("xprotect version && xprotect status"); err == nil {
		protectionSettings["xprotect"] = output
	}
	rawResults["protectionSettings"] = protectionSettings

	// Screen Lock Status
	screenLockStatus := make([]interface{}, 0)
	if result, err := c.RunQuery("SELECT value FROM preferences WHERE domain='com.apple.screensaver' AND key='idleTime' UNION ALL SELECT value FROM managed_policies WHERE domain='com.apple.screensaver' AND name='idleTime'"); err == nil {
		screenLockStatus = append(screenLockStatus, result)
	}
	if result, err := c.RunQuery("SELECT enabled, grace_period FROM screenlock"); err == nil {
		screenLockStatus = append(screenLockStatus, result)
	}
	rawResults["screenLockStatus"] = screenLockStatus

	// Screen Lock Settings
	screenLockSettings := make(map[string]interface{})
	if result, err := c.queryFirst("SELECT MAX(CAST(value AS INT)) AS value FROM preferences WHERE domain='com.apple.screensaver' AND key='idleTime' AND value IS NOT NULL AND host = 'current'"); err == nil && result != nil {
		screenLockSettings["screenSaverIdleWait"] = result["value"]
	}
	if output, err := c.RunCommand("pmset -g custom"); err == nil {
		screenLockSettings["powerSettings"] = output
	}
	if result, err := c.queryFirst("SELECT enabled, grace_period FROM screenlock"); err == nil && result != nil {
		screenLockSettings["lockDelay"] = result["grace_period"]
		screenLockSettings["screenLockEnabled"] = result["enabled"] == "1"
	}
	rawResults["screenLockSettings"] = screenLockSettings

	return &QueryResult{
		DrataAgentVersion: version,
		Platform:          PlatformMacOS,
		RawQueryResults:   rawResults,
	}, nil
}

// getMacOSDeviceIdentifiers returns macOS device identifiers.
func (c *Client) getMacOSDeviceIdentifiers() (*AgentDeviceIdentifiers, error) {
	identifiers := &AgentDeviceIdentifiers{}

	if result, err := c.queryFirst("SELECT hardware_serial, board_serial FROM system_info"); err == nil && result != nil {
		if v, ok := result["hardware_serial"].(string); ok {
			identifiers.HWSerial.HardwareSerial = v
		}
		if v, ok := result["board_serial"].(string); ok {
			identifiers.HWSerial.BoardSerial = v
		}
	}

	if result, err := c.queryFirst("SELECT mac FROM interface_details WHERE interface in (SELECT DISTINCT interface FROM interface_addresses WHERE interface IN ('en0', 'en1')) LIMIT 1"); err == nil && result != nil {
		if v, ok := result["mac"].(string); ok {
			identifiers.MacAddress.Mac = v
		}
	}

	return identifiers, nil
}

// containsIgnoreCase checks if a string contains a substring (case insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsLower(toLower(s), toLower(substr)))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
