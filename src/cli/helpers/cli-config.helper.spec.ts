import fs from 'fs';
import os from 'os';
import path from 'path';
import { CliConfigHelper } from './cli-config.helper';

// Mock fs and os modules
jest.mock('fs');
jest.mock('os');

describe('CliConfigHelper', () => {
    const mockHomedir = '/mock/home';
    const mockConfigPath = '/mock/home/.drata-agent/config.json';

    beforeEach(() => {
        // Reset singleton instance
        (CliConfigHelper as any)._instance = undefined;

        // Mock os.homedir
        (os.homedir as jest.Mock).mockReturnValue(mockHomedir);

        // Mock fs functions
        (fs.existsSync as jest.Mock).mockImplementation((path: string) => {
            if (path === '/mock/home/.drata-agent') return true;
            if (path === mockConfigPath) return true;
            return false;
        });

        (fs.readFileSync as jest.Mock).mockReturnValue(
            JSON.stringify({ region: 'NA', appVersion: '1.0.0' })
        );

        (fs.writeFileSync as jest.Mock).mockImplementation(() => {});
        (fs.mkdirSync as jest.Mock).mockImplementation(() => {});
    });

    afterEach(() => {
        jest.clearAllMocks();
    });

    it('should return the correct config path', () => {
        const helper = CliConfigHelper.instance;
        expect(helper.getConfigPath()).toBe(mockConfigPath);
    });

    it('should get a stored value', () => {
        const helper = CliConfigHelper.instance;
        expect(helper.get('region')).toBe('NA');
    });

    it('should set a value and write to config', () => {
        const helper = CliConfigHelper.instance;
        helper.set('appVersion', '2.0.0');

        expect(fs.writeFileSync).toHaveBeenCalled();
        expect(helper.get('appVersion')).toBe('2.0.0');
    });

    it('should check registration status', () => {
        (fs.readFileSync as jest.Mock).mockReturnValue(
            JSON.stringify({ accessToken: 'test-token' })
        );

        // Reset singleton to pick up new mock
        (CliConfigHelper as any)._instance = undefined;

        const helper = CliConfigHelper.instance;
        expect(helper.isRegistered).toBe(true);
    });

    it('should report not registered without access token', () => {
        (fs.readFileSync as jest.Mock).mockReturnValue(JSON.stringify({}));

        // Reset singleton to pick up new mock
        (CliConfigHelper as any)._instance = undefined;

        const helper = CliConfigHelper.instance;
        expect(helper.isRegistered).toBe(false);
    });

    it('should clear data', () => {
        const helper = CliConfigHelper.instance;
        helper.clearData();

        expect(fs.writeFileSync).toHaveBeenCalled();
    });

    it('should create config directory if it does not exist', () => {
        (fs.existsSync as jest.Mock).mockImplementation(() => false);

        // Reset singleton
        (CliConfigHelper as any)._instance = undefined;

        const helper = CliConfigHelper.instance;

        expect(fs.mkdirSync).toHaveBeenCalledWith(
            '/mock/home/.drata-agent',
            expect.objectContaining({ recursive: true })
        );
    });
});
