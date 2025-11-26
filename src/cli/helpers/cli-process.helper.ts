import { exec, execFile, ExecFileOptions } from 'child_process';
import { isEmpty, trim } from 'lodash';
import { CliLogger } from './cli-logger.helper';

/**
 * CLI-specific process helper that doesn't depend on Electron
 */
export class CliProcessHelper {
    private static readonly logger = new CliLogger('CliProcessHelper');

    static async runQuery(osqueryiBinaryPath: string, query: string): Promise<string> {
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
