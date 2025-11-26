# Drata Agent

The Drata Agent is a lightweight application that lives in your computer's toolbar. This application is granted READ ONLY access to your system preferences to help ensure proper security configurations are set - such as screensaver locking, password manager, antivirus software and automatic updates are enabled. These security configurations are required for SOC 2 compliance.

## CLI Version

A command-line interface (CLI) version of the Drata Agent is available in the `cli/` directory. The CLI version is written in Go and can be compiled for any platform. See [cli/README.md](cli/README.md) for more information.

### Quick Start (CLI)

```bash
cd cli

# Build
make build

# Register
./build/drata-agent register YOUR_TOKEN --region NA

# Sync
./build/drata-agent sync

# Run as daemon
./build/drata-agent daemon
```

## See also

- [CLI Documentation](cli/README.md)
- [Drata Help](https://help.drata.com/)
- [Automatic Upgrades Channel Repository](https://github.com/drata/agent-releases)
- [Electron](https://www.electronjs.org/)
- [Electron Builder](https://www.electron.build/)
- [osquery](https://www.osquery.io/)

# Linux CLI Application

For Linux users who prefer a command-line interface or need to run the agent in headless environments, a CLI version is available.

## Build CLI

```bash
# Install dependencies
yarn install --ignore-engines

# Build the CLI for production
yarn build:cli

# The CLI binary will be at dist/cli.js
```

## CLI Usage

The CLI stores configuration in `~/.drata-agent/config.json`.

### Available Commands

```bash
# Configure and register the agent (interactive)
node dist/cli.js configure

# Run a manual sync
node dist/cli.js sync

# Show agent status
node dist/cli.js status

# Show debug information
node dist/cli.js debug

# Disconnect and clear configuration
node dist/cli.js disconnect

# Show help
node dist/cli.js --help

# Show version
node dist/cli.js --version
```

### First-time Setup

1. Build the CLI using `yarn build:cli`
2. Run `node dist/cli.js configure`
3. Enter your registration token from Drata (found in My Drata → Install the Drata Agent → Register Drata Agent)
4. Select your region (NA, EU, or APAC)

### Running as a Scheduled Task

To run the CLI periodically (e.g., via cron), use the sync command:

```bash
# Example crontab entry to sync every 4 hours
0 */4 * * * /usr/bin/node /path/to/dist/cli.js sync
```

### Environment Variables

- `TARGET_ENV`: Target environment (LOCAL, DEV, QA, PROD). Defaults to PROD.
- `DEBUG`: Set to any value to enable debug logging.
- `CLI_PACKAGED`: Set to 'true' when running from a packaged binary.
- `CLI_OSQUERYI_PATH`: Custom path to osqueryi binary.

# Run or Build Drata Agent on Mac

## Caveats

- The Drata Agent requires an active production account to register successfully.
- Support is not provided for building, running, or installing unofficial packages.
- The build process outlined does not include secure code signing.
- IMPORTANT: Component Library Package is NOT provided. At this time, certain front end components will need replaced to build.
- IMPORTANT: osquery binaries are not tracked, they may be downloaded from release assets or directly from osquery.

## Prerequisites

1. XCode (command line tools)
1. NodeJS

## Run Local

```bash
# Run on local in dev mode
yarn start
```

## Build Package

The following commands will bundle and build a installation package into the local ./dist folder.

```bash
# Bundle
node_modules/.bin/webpack --mode=production --env targetEnv=PROD

# Build with profile - see package.json for configured profiles
node_modules/.bin/electron-builder --mac -c.mac.identity=null
```

## Install and register agent from local build

1. Switch/checkout this repository
1. Build desired package
1. Execute dmg disk image (dist folder) and copy Drata Agent to `Applications`
1. Run Drata Agent from `Applications`
1. Click Agent -> Settings Icon -> you can view which version of the agent is running, it should say `[LOCAL] Agent Version`
1. Log into Drata -> MyDrata -> Install the Drata Agent -> click Register Drata Agent. This will send a magic-link.
1. From the magic-link email, copy the token portion of the magic-link URL and paste it into your local Agent -> Click Register.

## Drata Support

Drata supports the latest and previous major LTS versions of Ubuntu, Windows, and macOS.

Please see Drata help for the most up-to-date support information.
