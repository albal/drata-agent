import { isNil } from 'lodash';
import { CliAxiosHelper } from '../helpers/cli-axios.helper';
import { CliConfigHelper } from '../helpers/cli-config.helper';
import { CliLogger } from '../helpers/cli-logger.helper';
import { AgentDeviceIdentifiers, QueryResult } from './cli-system-query.service';

interface AuthResponse {
    accessToken: string;
}

interface MeResponseDto {
    id: number;
    entryId: string;
    email: string;
    firstName: string;
    lastName: string;
    jobTitle?: string;
    avatarUrl: string;
    roles: string[];
    drataTermsAgreedAt: string;
    createdAt: string;
    updatedAt: string;
    signature: string;
    language: string;
}

interface AgentV2Response {
    lastCheckedAt?: string;
}

interface AgentV2ResponseDto {
    complianceChecks: any[];
    data: {
        lastcheckedAt: string;
    };
    winAvServicesMatchList: string[];
}

interface AgentInitializationDataResponseDto {
    winAvServicesMatchList: string[];
}

/**
 * CLI-specific API service that doesn't depend on Electron
 */
export class CliApiService {
    private readonly logger = new CliLogger('CliApiService');
    private readonly configHelper = CliConfigHelper.instance;
    private readonly axiosHelper = CliAxiosHelper.instance;

    async register(agentDeviceIdentifiers: AgentDeviceIdentifiers): Promise<void> {
        const { data } = await this.registerWorkstation(agentDeviceIdentifiers);

        if (isNil(data)) {
            throw new Error('Missing data on register request.');
        }

        const { lastCheckedAt } = data;
        this.configHelper.set('lastCheckedAt', lastCheckedAt);
    }

    async loginWithMagicLink(token: string): Promise<MeResponseDto> {
        const { data } = await this.authMagicLink(token);

        if (isNil(data)) {
            throw new Error('Missing data from auth request.');
        }

        if (!isNil(data.accessToken)) {
            this.configHelper.set('accessToken', data.accessToken);
        } else {
            this.configHelper.remove('accessToken');
            this.configHelper.remove('user');
        }

        // Once the accessToken is set we can start making authorized requests
        const { data: user } = await this.getMe();

        if (isNil(user)) {
            this.configHelper.clearData();
            this.configHelper.remove('accessToken');
            this.configHelper.remove('user');
            throw new Error('Missing data from get user info request.');
        }

        // Filter user data to only include fields we want
        const filteredUser = {
            id: user.id,
            email: user.email,
            firstName: user.firstName,
            lastName: user.lastName,
        };

        this.configHelper.set('user', filteredUser);

        return user;
    }

    async sync(results: QueryResult): Promise<AgentV2ResponseDto> {
        const { data } = await this.setPersonnelChecks(results);

        if (isNil(data)) {
            throw new Error('Missing data on set personnel checks request.');
        }

        this.configHelper.multiSet({
            lastCheckedAt: data.data.lastcheckedAt,
            winAvServicesMatchList: data.winAvServicesMatchList,
        });

        return data;
    }

    async initialData(): Promise<AgentInitializationDataResponseDto> {
        const { data } = await this.getInitData();

        if (isNil(data)) {
            throw new Error('Missing data on get initialization data request.');
        }

        this.configHelper.multiSet({
            winAvServicesMatchList: data.winAvServicesMatchList,
        });

        return data;
    }

    private authMagicLink(token: string) {
        return this.axiosHelper.post<AuthResponse>(`/auth/magic-link/${token}`);
    }

    private registerWorkstation(agentDeviceIdentifiers: AgentDeviceIdentifiers) {
        return this.axiosHelper.post<AgentV2Response>('/agentv2/register', agentDeviceIdentifiers);
    }

    private setPersonnelChecks(data: QueryResult) {
        return this.axiosHelper.post<AgentV2ResponseDto>('/agentv2/sync', data);
    }

    private getMe() {
        return this.axiosHelper.get<MeResponseDto>('/users/me');
    }

    private getInitData() {
        return this.axiosHelper.get<AgentInitializationDataResponseDto>('/agentv2/init');
    }
}
