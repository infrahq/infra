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

import { ObservableAuthApi } from "./ObservableAPI";
import { AuthApiRequestFactory, AuthApiResponseProcessor} from "../apis/AuthApi";

export interface AuthApiLoginRequest {
    /**
     * 
     * @type LoginRequest
     * @memberof AuthApilogin
     */
    body: LoginRequest
}

export interface AuthApiLogoutRequest {
}

export interface AuthApiSignupRequest {
    /**
     * 
     * @type SignupRequest
     * @memberof AuthApisignup
     */
    body: SignupRequest
}

export class ObjectAuthApi {
    private api: ObservableAuthApi

    public constructor(configuration: Configuration, requestFactory?: AuthApiRequestFactory, responseProcessor?: AuthApiResponseProcessor) {
        this.api = new ObservableAuthApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * Log in to Infra and get an API token for a user
     * @param param the request object
     */
    public login(param: AuthApiLoginRequest, options?: Configuration): Promise<AuthResponse> {
        return this.api.login(param.body,  options).toPromise();
    }

    /**
     * Log out of Infra
     * @param param the request object
     */
    public logout(param: AuthApiLogoutRequest, options?: Configuration): Promise<void> {
        return this.api.logout( options).toPromise();
    }

    /**
     * Sign up Infra's admin user and get an API token for a user
     * @param param the request object
     */
    public signup(param: AuthApiSignupRequest, options?: Configuration): Promise<AuthResponse> {
        return this.api.signup(param.body,  options).toPromise();
    }

}

import { ObservableCredsApi } from "./ObservableAPI";
import { CredsApiRequestFactory, CredsApiResponseProcessor} from "../apis/CredsApi";

export interface CredsApiCreateCredRequest {
}

export class ObjectCredsApi {
    private api: ObservableCredsApi

    public constructor(configuration: Configuration, requestFactory?: CredsApiRequestFactory, responseProcessor?: CredsApiResponseProcessor) {
        this.api = new ObservableCredsApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * Create credentials to access a destination
     * @param param the request object
     */
    public createCred(param: CredsApiCreateCredRequest, options?: Configuration): Promise<Cred> {
        return this.api.createCred( options).toPromise();
    }

}

import { ObservableDestinationsApi } from "./ObservableAPI";
import { DestinationsApiRequestFactory, DestinationsApiResponseProcessor} from "../apis/DestinationsApi";

export interface DestinationsApiCreateDestinationRequest {
    /**
     * 
     * @type DestinationCreateRequest
     * @memberof DestinationsApicreateDestination
     */
    body: DestinationCreateRequest
}

export interface DestinationsApiListDestinationsRequest {
}

export class ObjectDestinationsApi {
    private api: ObservableDestinationsApi

    public constructor(configuration: Configuration, requestFactory?: DestinationsApiRequestFactory, responseProcessor?: DestinationsApiResponseProcessor) {
        this.api = new ObservableDestinationsApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * Register a destination
     * @param param the request object
     */
    public createDestination(param: DestinationsApiCreateDestinationRequest, options?: Configuration): Promise<Destination> {
        return this.api.createDestination(param.body,  options).toPromise();
    }

    /**
     * List destinations
     * @param param the request object
     */
    public listDestinations(param: DestinationsApiListDestinationsRequest, options?: Configuration): Promise<Array<Destination>> {
        return this.api.listDestinations( options).toPromise();
    }

}

import { ObservableGroupsApi } from "./ObservableAPI";
import { GroupsApiRequestFactory, GroupsApiResponseProcessor} from "../apis/GroupsApi";

export interface GroupsApiListGroupsRequest {
}

export class ObjectGroupsApi {
    private api: ObservableGroupsApi

    public constructor(configuration: Configuration, requestFactory?: GroupsApiRequestFactory, responseProcessor?: GroupsApiResponseProcessor) {
        this.api = new ObservableGroupsApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * List groups
     * @param param the request object
     */
    public listGroups(param: GroupsApiListGroupsRequest, options?: Configuration): Promise<Array<Group>> {
        return this.api.listGroups( options).toPromise();
    }

}

import { ObservableInfoApi } from "./ObservableAPI";
import { InfoApiRequestFactory, InfoApiResponseProcessor} from "../apis/InfoApi";

export interface InfoApiStatusRequest {
}

export interface InfoApiVersionRequest {
}

export class ObjectInfoApi {
    private api: ObservableInfoApi

    public constructor(configuration: Configuration, requestFactory?: InfoApiRequestFactory, responseProcessor?: InfoApiResponseProcessor) {
        this.api = new ObservableInfoApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * Get signup status information
     * @param param the request object
     */
    public status(param: InfoApiStatusRequest, options?: Configuration): Promise<StatusResponse> {
        return this.api.status( options).toPromise();
    }

    /**
     * Get version information
     * @param param the request object
     */
    public version(param: InfoApiVersionRequest, options?: Configuration): Promise<VersionResponse> {
        return this.api.version( options).toPromise();
    }

}

import { ObservableRolesApi } from "./ObservableAPI";
import { RolesApiRequestFactory, RolesApiResponseProcessor} from "../apis/RolesApi";

export interface RolesApiListRolesRequest {
    /**
     * ID of the destination for which to list roles
     * @type string
     * @memberof RolesApilistRoles
     */
    destinationId: string
}

export class ObjectRolesApi {
    private api: ObservableRolesApi

    public constructor(configuration: Configuration, requestFactory?: RolesApiRequestFactory, responseProcessor?: RolesApiResponseProcessor) {
        this.api = new ObservableRolesApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * List roles
     * @param param the request object
     */
    public listRoles(param: RolesApiListRolesRequest, options?: Configuration): Promise<Array<Role>> {
        return this.api.listRoles(param.destinationId,  options).toPromise();
    }

}

import { ObservableSourcesApi } from "./ObservableAPI";
import { SourcesApiRequestFactory, SourcesApiResponseProcessor} from "../apis/SourcesApi";

export interface SourcesApiListSourcesRequest {
}

export class ObjectSourcesApi {
    private api: ObservableSourcesApi

    public constructor(configuration: Configuration, requestFactory?: SourcesApiRequestFactory, responseProcessor?: SourcesApiResponseProcessor) {
        this.api = new ObservableSourcesApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * List sources
     * @param param the request object
     */
    public listSources(param: SourcesApiListSourcesRequest, options?: Configuration): Promise<Array<Source>> {
        return this.api.listSources( options).toPromise();
    }

}

import { ObservableUsersApi } from "./ObservableAPI";
import { UsersApiRequestFactory, UsersApiResponseProcessor} from "../apis/UsersApi";

export interface UsersApiListUsersRequest {
}

export class ObjectUsersApi {
    private api: ObservableUsersApi

    public constructor(configuration: Configuration, requestFactory?: UsersApiRequestFactory, responseProcessor?: UsersApiResponseProcessor) {
        this.api = new ObservableUsersApi(configuration, requestFactory, responseProcessor);
    }

    /**
     * List users
     * @param param the request object
     */
    public listUsers(param: UsersApiListUsersRequest, options?: Configuration): Promise<Array<User>> {
        return this.api.listUsers( options).toPromise();
    }

}
