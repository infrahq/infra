import { ResponseContext, RequestContext, HttpFile } from '../http/http';
import * as models from '../models/all';
import { Configuration} from '../configuration'

import { AuthResponse } from '../models/AuthResponse';
import { Cred } from '../models/Cred';
import { Destination } from '../models/Destination';
import { DestinationCreateRequest } from '../models/DestinationCreateRequest';
import { DestinationKubernetes } from '../models/DestinationKubernetes';
import { Group } from '../models/Group';
import { LoginRequest } from '../models/LoginRequest';
import { LoginRequestInfra } from '../models/LoginRequestInfra';
import { LoginRequestOkta } from '../models/LoginRequestOkta';
import { ModelError } from '../models/ModelError';
import { Role } from '../models/Role';
import { RoleKind } from '../models/RoleKind';
import { SignupRequest } from '../models/SignupRequest';
import { Source } from '../models/Source';
import { SourceOkta } from '../models/SourceOkta';
import { StatusResponse } from '../models/StatusResponse';
import { User } from '../models/User';
import { VersionResponse } from '../models/VersionResponse';
import { ObservableAuthApi } from './ObservableAPI';

import { AuthApiRequestFactory, AuthApiResponseProcessor} from "../apis/AuthApi";
export class PromiseAuthApi {
    private api: ObservableAuthApi

    public constructor(
        configuration: Configuration,
        requestFactory?: AuthApiRequestFactory,
        responseProcessor?: AuthApiResponseProcessor
    ) {
        this.api = new ObservableAuthApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * Log in to Infra and get an API token for a user
     * @param body 
     */
    public login(body: LoginRequest, _options?: Configuration): Promise<AuthResponse> {
        const result = this.api.login(body, _options);
        return result.toPromise();
    }

    /**
     * Log out of Infra
     */
    public logout(_options?: Configuration): Promise<void> {
        const result = this.api.logout(_options);
        return result.toPromise();
    }

    /**
     * Sign up Infra's admin user and get an API token for a user
     * @param body 
     */
    public signup(body: SignupRequest, _options?: Configuration): Promise<AuthResponse> {
        const result = this.api.signup(body, _options);
        return result.toPromise();
    }


}



import { ObservableCredsApi } from './ObservableAPI';

import { CredsApiRequestFactory, CredsApiResponseProcessor} from "../apis/CredsApi";
export class PromiseCredsApi {
    private api: ObservableCredsApi

    public constructor(
        configuration: Configuration,
        requestFactory?: CredsApiRequestFactory,
        responseProcessor?: CredsApiResponseProcessor
    ) {
        this.api = new ObservableCredsApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * Create credentials to access a destination
     */
    public createCred(_options?: Configuration): Promise<Cred> {
        const result = this.api.createCred(_options);
        return result.toPromise();
    }


}



import { ObservableDestinationsApi } from './ObservableAPI';

import { DestinationsApiRequestFactory, DestinationsApiResponseProcessor} from "../apis/DestinationsApi";
export class PromiseDestinationsApi {
    private api: ObservableDestinationsApi

    public constructor(
        configuration: Configuration,
        requestFactory?: DestinationsApiRequestFactory,
        responseProcessor?: DestinationsApiResponseProcessor
    ) {
        this.api = new ObservableDestinationsApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * Register a destination
     * @param body 
     */
    public createDestination(body: DestinationCreateRequest, _options?: Configuration): Promise<Destination> {
        const result = this.api.createDestination(body, _options);
        return result.toPromise();
    }

    /**
     * List destinations
     */
    public listDestinations(_options?: Configuration): Promise<Array<Destination>> {
        const result = this.api.listDestinations(_options);
        return result.toPromise();
    }


}



import { ObservableGroupsApi } from './ObservableAPI';

import { GroupsApiRequestFactory, GroupsApiResponseProcessor} from "../apis/GroupsApi";
export class PromiseGroupsApi {
    private api: ObservableGroupsApi

    public constructor(
        configuration: Configuration,
        requestFactory?: GroupsApiRequestFactory,
        responseProcessor?: GroupsApiResponseProcessor
    ) {
        this.api = new ObservableGroupsApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * List groups
     */
    public listGroups(_options?: Configuration): Promise<Array<Group>> {
        const result = this.api.listGroups(_options);
        return result.toPromise();
    }


}



import { ObservableInfoApi } from './ObservableAPI';

import { InfoApiRequestFactory, InfoApiResponseProcessor} from "../apis/InfoApi";
export class PromiseInfoApi {
    private api: ObservableInfoApi

    public constructor(
        configuration: Configuration,
        requestFactory?: InfoApiRequestFactory,
        responseProcessor?: InfoApiResponseProcessor
    ) {
        this.api = new ObservableInfoApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * Get signup status information
     */
    public status(_options?: Configuration): Promise<StatusResponse> {
        const result = this.api.status(_options);
        return result.toPromise();
    }

    /**
     * Get version information
     */
    public version(_options?: Configuration): Promise<VersionResponse> {
        const result = this.api.version(_options);
        return result.toPromise();
    }


}



import { ObservableRolesApi } from './ObservableAPI';

import { RolesApiRequestFactory, RolesApiResponseProcessor} from "../apis/RolesApi";
export class PromiseRolesApi {
    private api: ObservableRolesApi

    public constructor(
        configuration: Configuration,
        requestFactory?: RolesApiRequestFactory,
        responseProcessor?: RolesApiResponseProcessor
    ) {
        this.api = new ObservableRolesApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * List roles
     * @param destinationId ID of the destination for which to list roles
     */
    public listRoles(destinationId: string, _options?: Configuration): Promise<Array<Role>> {
        const result = this.api.listRoles(destinationId, _options);
        return result.toPromise();
    }


}



import { ObservableSourcesApi } from './ObservableAPI';

import { SourcesApiRequestFactory, SourcesApiResponseProcessor} from "../apis/SourcesApi";
export class PromiseSourcesApi {
    private api: ObservableSourcesApi

    public constructor(
        configuration: Configuration,
        requestFactory?: SourcesApiRequestFactory,
        responseProcessor?: SourcesApiResponseProcessor
    ) {
        this.api = new ObservableSourcesApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * List sources
     */
    public listSources(_options?: Configuration): Promise<Array<Source>> {
        const result = this.api.listSources(_options);
        return result.toPromise();
    }


}



import { ObservableUsersApi } from './ObservableAPI';

import { UsersApiRequestFactory, UsersApiResponseProcessor} from "../apis/UsersApi";
export class PromiseUsersApi {
    private api: ObservableUsersApi

    public constructor(
        configuration: Configuration,
        requestFactory?: UsersApiRequestFactory,
        responseProcessor?: UsersApiResponseProcessor
    ) {
        this.api = new ObservableUsersApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * List users
     */
    public listUsers(_options?: Configuration): Promise<Array<User>> {
        const result = this.api.listUsers(_options);
        return result.toPromise();
    }


}



