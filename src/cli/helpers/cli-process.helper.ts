import { exec, execFile, ExecFileOptions } from 'child_process';
import { existsSync } from 'fs';
import path from 'path';
import { isEmpty, trim } from 'lodash';
import { CliLogger } from './cli-logger.helper';

/**
 * CLI-specific process helper that doesn't depend on Electron
 */
export class CliProcessHelper {
    private static readonly logger = new CliLogger('CliProcessHelper');

    /**
     * Validates that the osqueryi binary path is safe to execute
     * @param osqueryiBinaryPath Path to the osqueryi binary
     * @throws Error if the path is invalid or unsafe
     */
    private static validateOsqueryPath(osqueryiBinaryPath: string): void {
        // Normalize the path to resolve any .. or . components
        const normalizedPath = path.normalize(osqueryiBinaryPath);

        // Ensure the path ends with 'osqueryi' (case-sensitive on Linux)
        const basename = path.basename(normalizedPath);
        if (basename !== 'osqueryi') {
            throw new Error('Invalid osqueryi binary path: must end with "osqueryi"');
        }

        // Check for dangerous characters that could enable command injection
        if (/[;&|`$(){}[\]<>!]/.test(normalizedPath)) {
            throw new Error('Invalid osqueryi binary path: contains unsafe characters');
        }

        // Check that the file exists
        if (!existsSync(normalizedPath)) {
            throw new Error(`osqueryi binary not found at: ${normalizedPath}`);
        }
    }

    static async runQuery(osqueryiBinaryPath: string, query: string): Promise<string> {
        // Validate the osqueryi path before execution
        this.validateOsqueryPath(osqueryiBinaryPath);

        const result = await this.promiseExecFile(
            `"${osqueryiBinaryPath}"`,
            [query],
            {
                shell: true,
            }
        );

        return trim(result);
    }

    static async runCommand(command: string): Promise<string> {
        const result = await this.promiseExec(command);
        return trim(result);
    }

    private static promiseExec(command: string): Promise<string> {
        return new Promise((resolve, reject): void => {
            try {
                exec(command, (error, stdout, stderr) => {
                    if (error) {
                        return reject(error);
                    }

                    if (!isEmpty(stderr)) {
                        return reject(new Error(stderr.toString()));
                    }

                    resolve(stdout.toString());
                });
            } catch (error) {
                reject(error);
            }
        });
    }

    private static promiseExecFile(
        file: string,
        args: readonly string[] | null | undefined,
        options: ExecFileOptions
    ): Promise<string> {
        return new Promise((resolve, reject): void => {
            try {
                execFile(file, args, options, (error, stdout, stderr) => {
                    if (error) {
                        return reject(error);
                    }

                    if (!isEmpty(stderr)) {
                        this.logger.debug('Query info:', {
                            args,
                            info: stderr.toString(),
                        });
                    }

                    resolve(stdout.toString());
                });
            } catch (error) {
                reject(error);
            }
        });
    }
}
