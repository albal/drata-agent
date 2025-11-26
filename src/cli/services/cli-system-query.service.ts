import { flatten, isEmpty, isFunction, isNil } from 'lodash';
import path from 'path';
import { CliLogger } from '../helpers/cli-logger.helper';
import { CliProcessHelper } from '../helpers/cli-process.helper';

/**
 * Query type definition
 */
export interface Query {
    description: string;
    query?: string;
    command?: string;
    transform?: (result: any) => any;
}

/**
 * Query result type
 */
export interface QueryResult {
    drataAgentVersion: string;
    platform: 'MACOS' | 'WINDOWS' | 'LINUX';
    manualRun?: boolean;
    rawQueryResults: {
        osVersion: any;
        hwSerial: any;
        boardSerial?: any;
        boardModel?: any;
        computerName?: any;
        hostName?: any;
        localHostName?: any;
        hwModel: any;
        appList: any;
        firewallStatus: any;
        browserExtensions: any;
        macAddress: any;
        autoUpdateEnabled: any;
        autoUpdateSettings: any;
        screenLockStatus?: any;
        locationServices?: any;
        screenLockSettings?: any;
    };
}

/**
 * Agent device identifiers type
 */
export interface AgentDeviceIdentifiers {
    hwSerial: {
        hardware_serial: string | undefined;
        board_serial: string | undefined;
    };
    macAddress: { mac: string | undefined };
}

/**
 * CLI-specific Linux system query service that doesn't depend on Electron
 */
export class CliSystemQueryService {
    private readonly logger = new CliLogger('CliSystemQueryService');
    private readonly osqueryiBinaryPath: string;
    private readonly appVersion: string;

    static readonly SUPPORTED_SUSPEND_TYPES: string[] = ['hibernate', 'suspend'];

    constructor(appVersion: string) {
        this.appVersion = appVersion;

        // Determine osqueryi binary path
        // Priority: CLI_OSQUERYI_PATH env var > packaged location > development location
        if (process.env.CLI_OSQUERYI_PATH) {
            this.osqueryiBinaryPath = process.env.CLI_OSQUERYI_PATH;
        } else {
            // Use the lib directory path relative to the current working directory
            // This works both in development and when installed
            this.osqueryiBinaryPath = path.join(
                process.cwd(),
                'lib',
                'linux',
                'bin',
                'osqueryi'
            );
        }
    }

    async getSystemInfo(): Promise<QueryResult> {
        return {
            drataAgentVersion: this.appVersion,
            platform: 'LINUX',
            rawQueryResults: {
                osVersion: await this.runQuery({
                    description: "What Operating System is running and what is its version?",
                    query: 'SELECT name, version, platform FROM os_version',
                    transform: (res: any[]) => res[0],
                }),

                hwSerial: await this.runQuery({
                    description: "What is the workstation's serial number?",
                    query: 'SELECT hardware_serial FROM system_info',
                    transform: (res: any[]) => res[0],
                }),

                hwModel: await this.runQuery({
                    description: "What is the workstation's model?",
                    query: 'SELECT hardware_model FROM system_info',
                    transform: (res: any[]) => res[0],
                }),

                ...(await this.runQuery({
                    description: "What is the system information?",
                    query: 'SELECT board_serial, board_model, computer_name, hostname, local_hostname FROM system_info',
                    transform: (res: any[]) => ({
                        boardSerial: res[0]?.board_serial,
                        boardModel: res[0]?.board_model,
                        computerName: res[0]?.computer_name,
                        hostName: res[0]?.hostname,
                        localHostName: res[0]?.local_hostname,
                    }),
                })),

                firewallStatus: await this.runQuery({
                    description: 'Is the software firewall enabled on the workstation?',
                    query: "SELECT COUNT(*) AS passed FROM augeas WHERE path = '/etc/ufw/ufw.conf' AND label = 'ENABLED' AND value = 'yes'",
                    transform: (res: any[]) => res[0],
                }),

                appList: await this.runQuery({
                    description: 'Return a list of ALL applications installed on the workstation',
                    query: 'SELECT name, version FROM deb_packages',
                }),

                browserExtensions: await this.runQueries(
                    [
                        {
                            description: 'What are the Firefox extensions?',
                            query: 'SELECT name FROM firefox_addons',
                        },
                        {
                            description: 'What are the Chrome extensions?',
                            query: 'SELECT name FROM chrome_extensions',
                        },
                    ],
                    flatten
                ),

                macAddress: await this.runQuery({
                    description: 'What is the MAC Address of this machine?',
                    query: "SELECT mac FROM interface_details WHERE interface in (SELECT DISTINCT interface FROM interface_addresses WHERE interface NOT IN ('lo')) LIMIT 1",
                    transform: res => res[0],
                }),

                autoUpdateEnabled: await this.runQuery({
                    description: 'Is auto-update enabled on this machine?',
                    query: "SELECT COUNT(*) AS passed FROM file WHERE path = '/etc/apt/apt.conf.d/50unattended-upgrades'",
                    transform: (res: any[]) => res[0],
                }),

                autoUpdateSettings: await this.runQueries([
                    {
                        description: 'What are the automatic update settings?',
                        command: "apt-config dump | grep -E '^(APT::Periodic|Unattended-Upgrade)::'",
                    },
                    {
                        description: 'Are automatic updates scheduled?',
                        command: 'systemctl show apt-daily* --property=NextElapseUSecMonotonic,NextElapseUSecRealtime,Unit,Description,UnitFileState,LastTriggerUSec',
                    },
                    {
                        description: 'Have automatic updates had successes?',
                        command: 'journalctl -u apt-daily.service -u apt-daily-upgrade.service --since -7day -n 10 --no-pager --quiet',
                    },
                    {
                        description: 'Are any upgrades pending?',
                        command: '/usr/lib/update-notifier/apt-check',
                    },
                    {
                        description: 'When was the last update installed?',
                        command: "awk '/^Start-Date:/ {block=\"\"; inblock=1} inblock {block = block $0 ORS} /^End-Date:/ {if (block ~ /Upgrade:/) last=block; inblock=0} END {print last}' /var/log/apt/history.log",
                    },
                ]),

                screenLockStatus: await this.runQueries([
                    {
                        description: 'Time for screen to lock',
                        command: 'gsettings get org.gnome.desktop.screensaver lock-delay',
                    },
                    {
                        description: 'Is screenlock enabled?',
                        command: 'gsettings get org.gnome.desktop.screensaver lock-enabled',
                    },
                ]),

                locationServices: await this.runQuery({
                    description: 'Are location services enabled?',
                    command: 'gsettings get org.gnome.system.location enabled',
                    transform: (res: any) => ({ commandsResults: res }),
                }),

                screenLockSettings: {
                    ...(await this.runQueries(
                        [
                            {
                                description: 'Power settings',
                                command: 'gsettings list-recursively org.gnome.settings-daemon.plugins.power',
                            },
                            {
                                description: 'Screen saver settings',
                                command: 'gsettings list-recursively org.gnome.desktop.screensaver',
                            },
                            {
                                description: 'Session settings',
                                command: 'gsettings list-recursively org.gnome.desktop.session',
                            },
                        ],
                        res =>
                            this.processScreenSettings({
                                powerSettings: res?.[0],
                                screenSettings: res?.[1],
                                sessionSettings: res?.[2],
                            })
                    )),
                },
            },
        };
    }

    async getAgentDeviceIdentifiers(): Promise<AgentDeviceIdentifiers> {
        return {
            hwSerial: await this.runQuery({
                description: "What is the workstation's serial number?",
                query: 'SELECT hardware_serial, board_serial FROM system_info',
                transform: (res: any[]) => res[0],
            }),
            macAddress: await this.runQuery({
                description: 'What is the MAC Address of this machine?',
                query: "SELECT mac FROM interface_details WHERE interface in (SELECT DISTINCT interface FROM interface_addresses WHERE interface NOT IN ('lo')) LIMIT 1",
                transform: res => res[0],
            }),
        };
    }

    async getDebugInfo(): Promise<unknown> {
        return {
            osquery: await this.runQuery({
                description: 'What version of osquery are we using?',
                query: 'SELECT version from osquery_info',
                transform: (res: unknown[]) => res[0],
            }),
            os: await this.runQuery({
                description: 'What operating system and version are we using?',
                query: 'SELECT version, build, platform FROM os_version',
                transform: (res: unknown[]) => res[0],
            }),
            system_info: await this.getAgentDeviceIdentifiers(),
        };
    }

    private async runQueries<T>(
        queries?: Array<Query | undefined>,
        transform: (res: any) => T = (res: T) => res
    ): Promise<T | undefined> {
        let results: any;

        try {
            if (isNil(queries)) {
                return transform([]);
            }

            results = await Promise.all(queries.map(query => this.runQuery(query)));

            return transform(results);
        } catch (error) {
            this.logger.error(error, {
                message: `The queries "${queries?.map(query => query?.description).join(', ')} failed to run"`,
                rawResult: results,
            });

            return {} as T;
        }
    }

    private async runQuery(query?: Query): Promise<any | undefined> {
        let result: any;

        try {
            if (isNil(query)) {
                return;
            }

            if (typeof query.query !== 'undefined') {
                // Pass the query directly - execFile handles quoting safely
                const raw = await CliProcessHelper.runQuery(
                    this.osqueryiBinaryPath,
                    query.query
                );
                result = JSON.parse(raw);
            } else if (typeof query.command !== 'undefined') {
                result = await CliProcessHelper.runCommand(query.command);
            } else {
                throw new Error(
                    `The query "${query.description}" doesn't include either a query or a command.`
                );
            }

            if (isFunction(query.transform)) {
                return query.transform(result);
            }

            return result;
        } catch (error) {
            this.logger.error(error, {
                message: `The query "${query?.description}" failed to run "${query?.command ?? query?.query ?? 'ERROR: missing command and query'}"`,
                rawResult: result,
            });

            return {};
        }
    }

    private processScreenSettings({
        powerSettings,
        screenSettings,
        sessionSettings,
    }: {
        powerSettings: string | undefined;
        screenSettings: string | undefined;
        sessionSettings: string | undefined;
    }): any {
        try {
            const power = this.parseSettings(powerSettings) ?? {};
            const screen = this.parseSettings(screenSettings) ?? {};
            const session = this.parseSettings(sessionSettings) ?? {};

            return {
                screenLockEnabled: screen['lock-enabled'] === 'true',
                screenSaverIdleWait: this.parseIntOrUndefined(session['idle-delay']),
                lockDelay: this.parseIntOrUndefined(screen['lock-delay']),
                suspendScreenLockAC:
                    screen['ubuntu-lock-on-suspend'] === 'true' &&
                    CliSystemQueryService.SUPPORTED_SUSPEND_TYPES.includes(power['sleep-inactive-ac-type']),
                suspendScreenLockDC:
                    screen['ubuntu-lock-on-suspend'] === 'true' &&
                    CliSystemQueryService.SUPPORTED_SUSPEND_TYPES.includes(power['sleep-inactive-battery-type']),
                suspendIdleWaitAC: this.parseIntOrUndefined(power['sleep-inactive-ac-timeout']),
                suspendIdleWaitDC: this.parseIntOrUndefined(power['sleep-inactive-battery-timeout']),
            };
        } catch (error) {
            this.logger.error(error, 'Error processing settings.');
            return {};
        }
    }

    private parseSettings(settings: string | undefined): any {
        if (typeof settings !== 'string') return;
        return settings.split('\n').reduce((prev, cur) => {
            const item = cur.split(' ');
            return {
                ...prev,
                [item[1]]: item[item.length - 1],
            };
        }, {});
    }

    private parseIntOrUndefined(value: any): number | undefined {
        const ret = parseInt(value);
        return isNaN(ret) ? undefined : ret;
    }
}
