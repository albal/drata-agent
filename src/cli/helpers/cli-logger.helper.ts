/**
 * Simple CLI logger that writes to console and optionally to a log file
 */
export class CliLogger {
    constructor(private readonly context: string) {}

    info(...args: any[]): void {
        const message = this.formatMessage('INFO', args);
        console.log(message);
    }

    warn(...args: any[]): void {
        const message = this.formatMessage('WARN', args);
        console.warn(message);
    }

    error(error: any, context?: any): void {
        const errorMessage = error instanceof Error ? error.message : String(error);
        const message = this.formatMessage('ERROR', [errorMessage]);
        console.error(message);

        if (context) {
            console.error('Context:', JSON.stringify(context, null, 2));
        }
    }

    debug(...args: any[]): void {
        if (process.env.DEBUG) {
            const message = this.formatMessage('DEBUG', args);
            console.log(message);
        }
    }

    private formatMessage(level: string, args: any[]): string {
        const timestamp = new Date().toISOString();
        const formattedArgs = args
            .map(arg =>
                typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)
            )
            .join(' ');
        return `[${timestamp}] [${level}] <${this.context}>: ${formattedArgs}`;
    }
}
