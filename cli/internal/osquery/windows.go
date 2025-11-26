package osquery

import (
	"strings"
)

// getWindowsSystemInfo collects Windows-specific system information.
func (c *Client) getWindowsSystemInfo(version string) (*QueryResult, error) {
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

	// Firewall Status
	if result, err := c.queryFirst("SELECT firewall FROM windows_security_center"); err == nil && result != nil {
		rawResults["firewallStatus"] = result
	}

	// Application List
	if result, err := c.RunQuery("SELECT name, version FROM programs"); err == nil {
		rawResults["appList"] = result
	}

	// Browser Extensions
	extensions, _ := c.queryAll([]string{
		"SELECT name FROM firefox_addons",
		"SELECT name FROM chrome_extensions",
		"SELECT name FROM ie_extensions",
	})
	rawResults["browserExtensions"] = extensions

	// MAC Address
	if result, err := c.queryFirst("SELECT mac FROM interface_details WHERE physical_adapter=1"); err == nil && result != nil {
		rawResults["macAddress"] = result
	}

	// Auto Update
	if result, err := c.queryFirst("SELECT IIF(autoupdate == 'Good', 1, 0) AS autoUpdateEnabled FROM windows_security_center"); err == nil && result != nil {
		rawResults["autoUpdateEnabled"] = result["autoUpdateEnabled"] == "1"
	}

	// Screen Lock Status
	if output, err := c.RunCommand("powercfg /QH SCHEME_CURRENT SUB_VIDEO VIDEOCONLOCK 2> NUL && powercfg /QH SCHEME_CURRENT SUB_NONE CONSOLELOCK 2> NUL && powercfg /QH SCHEME_CURRENT SUB_SLEEP STANDBYIDLE 2> NUL"); err == nil {
		rawResults["screenLockStatus"] = map[string]interface{}{
			"commandResults": output,
		}
	}

	// Windows AV Status
	if result, err := c.queryFirst("SELECT antivirus FROM windows_security_center LIMIT 1"); err == nil && result != nil {
		rawResults["winAvStatus"] = result
	}

	// Windows Services List (filtered for AV services)
	if result, err := c.RunQuery("SELECT name, description, status, start_type FROM services"); err == nil {
		rawResults["winServicesList"] = result
	}

	// HDD Encryption Status (BitLocker)
	if output, err := c.RunCommand("powershell -NoProfile -command (New-Object -ComObject Shell.Application).NameSpace((Get-ChildItem Env:SystemDrive).Value).Self.ExtendedProperty('System.Volume.BitLockerProtection')"); err == nil {
		rawResults["hddEncryptionStatus"] = strings.TrimSpace(output)
	}

	// Screen Lock Settings
	screenLockSettings := make(map[string]interface{})

	// Screen saver settings from registry
	screenSaverQuery := `WITH policy_setting(pname, pdata) AS (
		SELECT name, MAX(CAST(data AS INT)) AS data FROM logon_sessions
		LEFT JOIN registry r2 ON r2.key = 'HKEY_USERS\' || logon_sid || '\SOFTWARE\Policies\Microsoft\Windows\Control Panel\Desktop'
		WHERE logon_type LIKE '%Interactive%' AND name IN ('ScreenSaveTimeOut', 'ScreenSaverIsSecure', 'ScreenSaveActive', 'DelayLockInterval')
		GROUP BY logon_sid, name
	), user_setting(uname, udata) AS (
		SELECT name, MAX(CAST(data AS INT)) AS data FROM logon_sessions
		JOIN registry ON key = 'HKEY_USERS\' || logon_sid || '\Control Panel\Desktop'
		WHERE logon_type LIKE '%Interactive%' AND name IN ('ScreenSaveTimeOut', 'ScreenSaverIsSecure', 'ScreenSaveActive', 'DelayLockInterval')
		GROUP BY logon_sid, name
	)
	SELECT COALESCE(pname, uname) AS name, COALESCE(pdata, udata) AS data FROM policy_setting
	FULL JOIN user_setting ON pname = uname`

	if result, err := c.RunQuery(screenSaverQuery); err == nil {
		settings := pivotResults(result)
		if screenSaverIsSecure, ok := settings["ScreenSaverIsSecure"]; ok {
			if screenSaveActive, ok := settings["ScreenSaveActive"]; ok {
				screenLockSettings["screenLockEnabled"] = screenSaverIsSecure == "1" && screenSaveActive == "1"
			}
		}
		if screenSaveTimeOut, ok := settings["ScreenSaveTimeOut"]; ok {
			screenLockSettings["screenSaverIdleWait"] = screenSaveTimeOut
		}
		if delayLockInterval, ok := settings["DelayLockInterval"]; ok {
			screenLockSettings["lockDelay"] = delayLockInterval
		}
	}

	// Machine inactivity limit policy
	if result, err := c.queryFirst("SELECT data FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System\\InactivityTimeoutSecs' COLLATE NOCASE"); err == nil && result != nil {
		screenLockSettings["machineInactivityLimit"] = result["data"]
	}

	rawResults["screenLockSettings"] = screenLockSettings

	return &QueryResult{
		DrataAgentVersion: version,
		Platform:          PlatformWindows,
		RawQueryResults:   rawResults,
	}, nil
}

// getWindowsDeviceIdentifiers returns Windows device identifiers.
func (c *Client) getWindowsDeviceIdentifiers() (*AgentDeviceIdentifiers, error) {
	identifiers := &AgentDeviceIdentifiers{}

	if result, err := c.queryFirst("SELECT hardware_serial, board_serial FROM system_info"); err == nil && result != nil {
		if v, ok := result["hardware_serial"].(string); ok {
			identifiers.HWSerial.HardwareSerial = v
		}
		if v, ok := result["board_serial"].(string); ok {
			identifiers.HWSerial.BoardSerial = v
		}
	}

	if result, err := c.queryFirst("SELECT mac FROM interface_details WHERE physical_adapter=1"); err == nil && result != nil {
		if v, ok := result["mac"].(string); ok {
			identifiers.MacAddress.Mac = v
		}
	}

	return identifiers, nil
}

// pivotResults converts name/data pairs to a map.
func pivotResults(results []map[string]interface{}) map[string]string {
	pivot := make(map[string]string)
	for _, row := range results {
		if name, ok := row["name"].(string); ok {
			if data, ok := row["data"].(string); ok {
				pivot[name] = data
			}
		}
	}
	return pivot
}
