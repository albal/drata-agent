import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import crypto from 'crypto';
import { Region } from '../../enums/region.enum';
import { TargetEnv } from '../../enums/target-env.enum';
import { CliConfigHelper } from './cli-config.helper';

const ApiHostUrl: Record<TargetEnv, Record<Region, string>> = {
    [TargetEnv.LOCAL]: {
        [Region.NA]: 'http://localhost:3000',
        [Region.EU]: 'http://localhost:3001',
        [Region.APAC]: 'http://localhost:3002',
    },
    [TargetEnv.PROD]: {
        [Region.NA]: 'https://agent.drata.com',
        [Region.EU]: 'https://agent.eu.drata.com',
        [Region.APAC]: 'https://agent.apac.drata.com',
    },
    [TargetEnv.DEV]: {
        [Region.NA]: 'https://agent.dev.drata.com',
        [Region.EU]: 'https://agent.dev.drata.com',
        [Region.APAC]: 'https://agent.dev.drata.com',
    },
    [TargetEnv.QA]: {
        [Region.NA]: 'https://agent.qa.drata.com',
        [Region.EU]: 'https://agent.qa.drata.com',
        [Region.APAC]: 'https://agent.qa.drata.com',
    },
};

/**
 * CLI-specific Axios helper that doesn't depend on Electron
 */
export class CliAxiosHelper {
    static readonly instance: CliAxiosHelper = new CliAxiosHelper();
    private readonly configHelper = CliConfigHelper.instance;
    private readonly axiosInstance: AxiosInstance;
    private readonly REQUEST_TIMEOUT = 5 * 60 * 1000;

    private constructor() {
        this.axiosInstance = axios.create();

        this.axiosInstance.interceptors.request.use(
            async config => {
                const region = this.configHelper.get('region');

                let uuid = this.configHelper.get('uuid');
                if (!uuid) {
                    uuid = crypto.randomUUID();
                    this.configHelper.set('uuid', uuid);
                }

                config.timeout = this.REQUEST_TIMEOUT;
                config.baseURL = this.resolveBaseUrl(region);
                config.headers.Authorization = `Bearer ${this.configHelper.get('accessToken')}`;
                config.headers['Content-Type'] = 'application/json';
                config.headers['Correlation-Id'] = uuid;
                config.headers['User-Agent'] = `Drata-Agent-CLI/${this.configHelper.get('appVersion') || '0.0.0'} (linux)`;

                return config;
            },
            error => Promise.reject(new Error(error))
        );
    }

    private resolveBaseUrl(region?: Region): string {
        const targetEnv = (process.env.TARGET_ENV as TargetEnv) || TargetEnv.PROD;
        const resolvedRegion = region || Region.NA;

        const hostUrl = ApiHostUrl[targetEnv]?.[resolvedRegion];
        if (!hostUrl) {
            throw new Error(`Unable to resolve region ${resolvedRegion} for environment ${targetEnv}.`);
        }

        return hostUrl;
    }

    get<T = any, R = AxiosResponse<T>>(
        url: string,
        config?: AxiosRequestConfig<any>
    ): Promise<R> {
        return this.axiosInstance.get(url, config);
    }

    post<T = any, R = AxiosResponse<T>>(
        url: string,
        data?: any,
        config?: AxiosRequestConfig<any>
    ): Promise<R> {
        return this.axiosInstance.post(url, data, config);
    }
}
