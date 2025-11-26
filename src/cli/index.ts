#!/usr/bin/env node

import yargs from 'yargs';
import { hideBin } from 'yargs/helpers';
import { select, input, confirm } from '@inquirer/prompts';
import { Spinner } from 'cli-spinner';
import chalk from 'chalk';
import { Region } from '../enums/region.enum';
import { SyncState } from '../enums/sync-state.enum';
import { CliConfigHelper } from './helpers/cli-config.helper';
import { CliLogger } from './helpers/cli-logger.helper';
import { CliApiService } from './services/cli-api.service';
import { CliSystemQueryService } from './services/cli-system-query.service';

const VERSION = '3.9.0';
const logger = new CliLogger('DrataAgentCLI');

/**
 * Configure the agent with interactive prompts
 */
async function configure(): Promise<void> {
    const configHelper = CliConfigHelper.instance;

    console.log(chalk.blue('\n🔧 Drata Agent CLI Configuration\n'));

    if (configHelper.isRegistered) {
        const user = configHelper.get('user');
        console.log(chalk.yellow(`Already registered as: ${user?.email || 'Unknown'}`));

        const reconfigure = await confirm({
            message: 'Do you want to reconfigure? This will disconnect the current device.',
            default: false,
        });

        if (!reconfigure) {
            console.log(chalk.green('Configuration unchanged.'));
            return;
        }

        configHelper.clearData();
        console.log(chalk.yellow('Previous configuration cleared.'));
    }

    // Get registration token
    console.log(chalk.gray('\nTo register the agent:'));
    console.log(chalk.gray('1. Log into Drata at https://app.drata.com'));
    console.log(chalk.gray('2. Go to My Drata → Install the Drata Agent'));
    console.log(chalk.gray('3. Click "Register Drata Agent" to get a registration token\n'));

    const token = await input({
        message: 'Enter your registration token:',
        validate: (value) => {
            if (!value || value.trim().length === 0) {
                return 'Token is required';
            }
            return true;
        },
    });

    // Select region
    const region = await select({
        message: 'Select your region:',
        choices: [
            { name: 'North America (NA)', value: Region.NA },
            { name: 'Europe (EU)', value: Region.EU },
            { name: 'Asia Pacific (APAC)', value: Region.APAC },
        ],
    });

    // Save region before attempting registration
    configHelper.set('region', region);
    configHelper.set('appVersion', VERSION);

    const spinner = new Spinner('Registering agent... %s');
    spinner.setSpinnerString('|/-\\');
    spinner.start();

    try {
        const apiService = new CliApiService();
        const systemQueryService = new CliSystemQueryService(VERSION);

        // Authenticate with the token
        await apiService.loginWithMagicLink(token.trim());

        // Register the device
        const deviceIdentifiers = await systemQueryService.getAgentDeviceIdentifiers();
        await apiService.register(deviceIdentifiers);

        spinner.stop(true);
        console.log(chalk.green('\n✓ Agent registered successfully!'));

        const user = configHelper.get('user');
        console.log(chalk.green(`  Email: ${user?.email || 'Unknown'}`));
        console.log(chalk.green(`  Region: ${region}`));
        console.log(chalk.gray(`  Config file: ${configHelper.getConfigPath()}\n`));
    } catch (error: any) {
        spinner.stop(true);
        console.log(chalk.red('\n✗ Registration failed:'));
        console.log(chalk.red(`  ${error.message || 'Unknown error'}`));
        
        // Clean up on failure
        configHelper.clearData();
        process.exit(1);
    }
}

/**
 * Run a sync operation
 */
async function sync(options: { verbose?: boolean }): Promise<void> {
    const configHelper = CliConfigHelper.instance;

    if (!configHelper.isRegistered) {
        console.log(chalk.red('Agent is not registered. Run "drata-agent configure" first.'));
        process.exit(1);
    }

    console.log(chalk.blue('\n🔄 Running sync...\n'));

    const spinner = new Spinner('Collecting system information... %s');
    spinner.setSpinnerString('|/-\\');
    spinner.start();

    try {
        const apiService = new CliApiService();
        const systemQueryService = new CliSystemQueryService(VERSION);

        // Check if we need initialization data
        if (!configHelper.isInitDataReady) {
            spinner.stop(true);
            console.log(chalk.gray('Fetching initialization data...'));
            await apiService.initialData();
            spinner.start();
        }

        configHelper.set('lastSyncAttemptedAt', new Date().toISOString());
        configHelper.set('syncState', SyncState.RUNNING);

        // Get system info
        spinner.stop(true);
        if (options.verbose) {
            console.log(chalk.gray('Querying system information...'));
        }
        
        const queryResults = await systemQueryService.getSystemInfo();
        queryResults.manualRun = true;

        if (options.verbose) {
            console.log(chalk.gray('System info collected. Sending to Drata...'));
        }

        spinner.setSpinnerTitle('Sending data to Drata... %s');
        spinner.start();

        // Send to API
        await apiService.sync(queryResults);

        configHelper.set('syncState', SyncState.SUCCESS);

        spinner.stop(true);
        console.log(chalk.green('✓ Sync completed successfully!'));
        console.log(chalk.gray(`  Last sync: ${configHelper.get('lastCheckedAt') || 'N/A'}\n`));
    } catch (error: any) {
        spinner.stop(true);
        configHelper.set('syncState', SyncState.ERROR);

        console.log(chalk.red('✗ Sync failed:'));
        console.log(chalk.red(`  ${error.message || 'Unknown error'}`));
        process.exit(1);
    }
}

/**
 * Show current status
 */
async function status(): Promise<void> {
    const configHelper = CliConfigHelper.instance;

    console.log(chalk.blue('\n📊 Drata Agent CLI Status\n'));

    console.log(`Version: ${chalk.cyan(VERSION)}`);
    console.log(`Config file: ${chalk.gray(configHelper.getConfigPath())}`);
    console.log();

    if (configHelper.isRegistered) {
        const user = configHelper.get('user');
        const region = configHelper.get('region');
        const syncState = configHelper.get('syncState');
        const lastCheckedAt = configHelper.get('lastCheckedAt');
        const lastSyncAttemptedAt = configHelper.get('lastSyncAttemptedAt');

        console.log(`Status: ${chalk.green('Registered')}`);
        console.log(`Email: ${chalk.cyan(user?.email || 'Unknown')}`);
        console.log(`Region: ${chalk.cyan(region || 'Unknown')}`);
        console.log(`Sync State: ${chalk.cyan(syncState || 'Unknown')}`);
        console.log(`Last Successful Sync: ${chalk.cyan(lastCheckedAt || 'Never')}`);
        console.log(`Last Sync Attempted: ${chalk.cyan(lastSyncAttemptedAt || 'Never')}`);
    } else {
        console.log(`Status: ${chalk.yellow('Not Registered')}`);
        console.log(chalk.gray('\nRun "drata-agent configure" to set up the agent.'));
    }

    console.log();
}

/**
 * Show debug information
 */
async function debug(): Promise<void> {
    const configHelper = CliConfigHelper.instance;

    console.log(chalk.blue('\n🔍 Debug Information\n'));

    const systemQueryService = new CliSystemQueryService(VERSION);

    const spinner = new Spinner('Collecting debug info... %s');
    spinner.setSpinnerString('|/-\\');
    spinner.start();

    try {
        const debugInfo = await systemQueryService.getDebugInfo();
        spinner.stop(true);

        console.log(chalk.gray('System Debug Information:'));
        console.log(JSON.stringify(debugInfo, null, 2));
        console.log();

        if (configHelper.isRegistered) {
            const user = configHelper.get('user');
            console.log(chalk.gray('Registration Info:'));
            console.log(`  Email: ${user?.email || 'Unknown'}`);
            console.log(`  Region: ${configHelper.get('region') || 'Unknown'}`);
        }

        console.log();
    } catch (error: any) {
        spinner.stop(true);
        console.log(chalk.red('Error collecting debug info:'));
        console.log(chalk.red(`  ${error.message || 'Unknown error'}`));
    }
}

/**
 * Disconnect and clear configuration
 */
async function disconnect(): Promise<void> {
    const configHelper = CliConfigHelper.instance;

    if (!configHelper.isRegistered) {
        console.log(chalk.yellow('Agent is not registered.'));
        return;
    }

    const user = configHelper.get('user');
    console.log(chalk.yellow(`\nCurrently registered as: ${user?.email || 'Unknown'}`));

    const confirmed = await confirm({
        message: 'Are you sure you want to disconnect this device?',
        default: false,
    });

    if (confirmed) {
        configHelper.clearData();
        console.log(chalk.green('\n✓ Device disconnected successfully.\n'));
    } else {
        console.log(chalk.gray('\nOperation cancelled.\n'));
    }
}

// Main CLI setup
yargs(hideBin(process.argv))
    .scriptName('drata-agent')
    .usage('$0 <command> [options]')
    .command(
        'configure',
        'Configure and register the Drata Agent',
        {},
        async () => {
            await configure();
        }
    )
    .command(
        'sync',
        'Manually run a sync operation',
        (yargs) => {
            return yargs.option('verbose', {
                alias: 'v',
                type: 'boolean',
                description: 'Show verbose output',
            });
        },
        async (argv) => {
            await sync({ verbose: argv.verbose });
        }
    )
    .command(
        'status',
        'Show current agent status',
        {},
        async () => {
            await status();
        }
    )
    .command(
        'debug',
        'Show debug information',
        {},
        async () => {
            await debug();
        }
    )
    .command(
        'disconnect',
        'Disconnect and clear configuration',
        {},
        async () => {
            await disconnect();
        }
    )
    .demandCommand(1, 'You need to specify a command')
    .strict()
    .version(VERSION)
    .alias('h', 'help')
    .alias('v', 'version')
    .help()
    .parse();
