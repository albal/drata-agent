# Drata Agent CLI

A command-line interface version of the Drata Agent that can be compiled for any platform. This CLI version provides the same compliance monitoring functionality as the desktop application but without the GUI components.

## Overview

The Drata Agent CLI is a lightweight command-line application that monitors your system's security configuration for SOC 2 compliance. It collects read-only information about your system's security settings including:

- Screensaver locking configuration
- Password manager detection
- Antivirus software status
- Automatic updates settings
- Disk encryption status
- Firewall configuration
- Installed applications
- Browser extensions

## Prerequisites

1. **osquery**: The CLI requires osquery to be installed on your system.
   - Download from: https://osquery.io/
   - Or install via package manager:
     - macOS: `brew install osquery`
     - Ubuntu: `apt install osquery`
     - Windows: Download from osquery.io

2. **Go 1.21+** (for building from source)

## Installation

### From Source

```bash
cd cli
make build
```

This creates the binary in `cli/build/drata-agent`.

### Cross-Platform Builds

Build for all supported platforms:

```bash
make build-all
```

This creates binaries for:
- Linux (amd64, arm64)
- macOS (amd64, arm64/Apple Silicon)
- Windows (amd64)

## Usage

### Register the Agent

First, obtain a registration token from Drata:
1. Log in to Drata at https://app.drata.com
2. Go to My Drata > Install the Drata Agent
3. Click "Register Drata Agent"
4. Copy the token from the magic link URL

Then register the agent:

```bash
drata-agent register YOUR_TOKEN --region NA
```

Available regions: `NA` (North America), `EU` (Europe), `APAC` (Asia-Pacific)

### Sync System Information

Manually sync your system information:

```bash
drata-agent sync
```

Force sync (ignoring throttle limits):

```bash
drata-agent sync --force
```

### Check Status

View the current agent status:

```bash
drata-agent status

# With detailed system information
drata-agent status --verbose
```

### Run as Daemon

Run the agent as a background daemon with periodic syncs:

```bash
drata-agent daemon

# With custom sync interval (hours)
drata-agent daemon --interval 4
```

The daemon can be managed with systemd, launchd, or Windows services.

### Configuration

View current configuration:

```bash
drata-agent config show
```

Set configuration values:

```bash
drata-agent config set region EU
drata-agent config set sync_interval_hours 4
```

Initialize configuration file with defaults:

```bash
drata-agent config init
```

View configuration file path:

```bash
drata-agent config path
```

### Unregister

Remove registration from this device:

```bash
drata-agent unregister
```

## Configuration

Configuration is stored in `$HOME/.drata-agent/config.yaml`.

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `region` | Drata region (NA, EU, APAC) | NA |
| `target_env` | Target environment (PROD, DEV, QA, LOCAL) | PROD |
| `sync_interval_hours` | Hours between automatic syncs | 2 |
| `min_hours_since_last_sync` | Minimum hours between syncs | 24 |
| `min_minutes_between_syncs` | Minimum minutes between sync attempts | 15 |
| `osquery_path` | Path to osquery binary (empty for auto-detect) | (auto) |

### Environment Variables

All configuration options can be set via environment variables with the `DRATA_` prefix:

```bash
export DRATA_REGION=EU
export DRATA_SYNC_INTERVAL_HOURS=4
```

## Running as a Service

### Linux (systemd)

Create `/etc/systemd/system/drata-agent.service`:

```ini
[Unit]
Description=Drata Agent
After=network.target

[Service]
Type=simple
User=your-username
ExecStart=/usr/local/bin/drata-agent daemon
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable drata-agent
sudo systemctl start drata-agent
```

### macOS (launchd)

Create `~/Library/LaunchAgents/com.drata.agent.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.drata.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/drata-agent</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load and start:

```bash
launchctl load ~/Library/LaunchAgents/com.drata.agent.plist
```

## Data Storage

Agent data is stored in `$HOME/.drata-agent/data/`:
- `app-data.json` - Registration and sync state

## Troubleshooting

### osquery not found

If you get an error about osquery not being found:

1. Install osquery from https://osquery.io/
2. Or specify the path manually:
   ```bash
   drata-agent config set osquery_path /path/to/osqueryi
   ```

### Authentication errors

If you get authentication errors:
1. Unregister: `drata-agent unregister`
2. Get a new registration token from Drata
3. Register again: `drata-agent register NEW_TOKEN --region NA`

### View logs

When running as a daemon, logs are written to stdout. Redirect to a file if needed:

```bash
drata-agent daemon 2>&1 | tee -a /var/log/drata-agent.log
```

## Support

For support, please visit https://help.drata.com/

## License

Apache-2.0
