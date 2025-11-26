import fs from 'fs';
import path from 'path';
import os from 'os';
import { Region } from '../../enums/region.enum';
import { SyncState } from '../../enums/sync-state.enum';

export interface CliConfig {
    uuid?: string;
    appVersion?: string;
    accessToken?: string;
    user?: {
        id: number;
        email: string;
        firstName: string;
        lastName: string;
    };
    syncState?: SyncState;
    lastCheckedAt?: string;
    lastSyncAttemptedAt?: string;
    winAvServicesMatchList?: string[];
    region?: Region;
}

/**
 * CLI Configuration helper that stores configuration in a plain text file
 * in the user's home directory (~/.drata-agent/config.json)
 */
export class CliConfigHelper {
    private static _instance: CliConfigHelper;
    private readonly configDir: string;
    private readonly configPath: string;
    private config: CliConfig;

    static get instance(): CliConfigHelper {
        if (!this._instance) {
            this._instance = new CliConfigHelper();
        }
        return this._instance;
    }

    private constructor() {
        this.configDir = path.join(os.homedir(), '.drata-agent');
        this.configPath = path.join(this.configDir, 'config.json');

        this.ensureConfigDir();
        this.config = this.loadConfig();
    }

    /**
     * Get configuration file path
     */
    getConfigPath(): string {
        return this.configPath;
    }

    /**
     * Retrieves a value from the configuration
     * @param key Configuration key
     * @returns The value for the key
     */
    get<K extends keyof CliConfig>(key: K): CliConfig[K] {
        return this.config[key];
    }

    /**
     * Stores a value in the configuration
     * @param key Configuration key
     * @param val Value to store
     */
    set<K extends keyof CliConfig>(key: K, val: CliConfig[K]): void {
        this.config[key] = val;
        this.writeConfig();
    }

    /**
     * Stores multiple values in the configuration
     * @param update Partial configuration to merge
     */
    multiSet(update: Partial<CliConfig>): void {
        this.config = {
            ...this.config,
            ...update,
        };
        this.writeConfig();
    }

    /**
     * Deletes a value from the configuration
     * @param key Configuration key to remove
     */
    remove<K extends keyof CliConfig>(key: K): void {
        delete this.config[key];
        this.writeConfig();
    }

    /**
     * Clears all configuration data
     */
    clearData(): void {
        this.config = {};
        this.writeConfig();
    }

    /**
     * Check if initialization data is ready
     */
    get isInitDataReady(): boolean {
        return this.config.winAvServicesMatchList !== undefined;
    }

    /**
     * Check if the agent is registered (has an access token)
     */
    get isRegistered(): boolean {
        return !!this.config.accessToken;
    }

    private ensureConfigDir(): void {
        if (!fs.existsSync(this.configDir)) {
            fs.mkdirSync(this.configDir, { recursive: true, mode: 0o700 });
        }
    }

    private loadConfig(): CliConfig {
        try {
            if (fs.existsSync(this.configPath)) {
                const data = fs.readFileSync(this.configPath, { encoding: 'utf8' });
                return JSON.parse(data) as CliConfig;
            }
        } catch (error) {
            console.error('Error loading configuration:', error);
        }
        return {};
    }

    private writeConfig(): void {
        try {
            fs.writeFileSync(this.configPath, JSON.stringify(this.config, null, 4), {
                encoding: 'utf8',
                mode: 0o600,
            });
        } catch (error) {
            console.error('Error writing configuration:', error);
            throw error;
        }
    }
}
