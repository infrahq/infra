/* tslint:disable */
/* eslint-disable */
/**
 * Infra API
 * Infra REST API
 *
 * The version of the OpenAPI document: 0.1.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */


import * as runtime from '../runtime';
import {
    AuthResponse,
    AuthResponseFromJSON,
    AuthResponseToJSON,
    LoginRequest,
    LoginRequestFromJSON,
    LoginRequestToJSON,
} from '../models';

export interface LoginOperationRequest {
    body: LoginRequest;
}

/**
 * 
 */
export class AuthApi extends runtime.BaseAPI {

    /**
     * Login to Infra and get an API token for a user
     */
    async loginRaw(requestParameters: LoginOperationRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<AuthResponse>> {
        if (requestParameters.body === null || requestParameters.body === undefined) {
            throw new runtime.RequiredError('body','Required parameter requestParameters.body was null or undefined when calling login.');
        }

        const queryParameters: any = {};

        const headerParameters: runtime.HTTPHeaders = {};

        headerParameters['Content-Type'] = 'application/json';

        const response = await this.request({
            path: `/login`,
            method: 'POST',
            headers: headerParameters,
            query: queryParameters,
            body: LoginRequestToJSON(requestParameters.body),
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => AuthResponseFromJSON(jsonValue));
    }

    /**
     * Login to Infra and get an API token for a user
     */
    async login(requestParameters: LoginOperationRequest, initOverrides?: RequestInit): Promise<AuthResponse> {
        const response = await this.loginRaw(requestParameters, initOverrides);
        return await response.value();
    }

    /**
     * Log out of Infra
     */
    async logoutRaw(initOverrides?: RequestInit): Promise<runtime.ApiResponse<void>> {
        const queryParameters: any = {};

        const headerParameters: runtime.HTTPHeaders = {};

        if (this.configuration && this.configuration.accessToken) {
            const token = this.configuration.accessToken;
            const tokenString = await token("bearerAuth", []);

            if (tokenString) {
                headerParameters["Authorization"] = `Bearer ${tokenString}`;
            }
        }
        const response = await this.request({
            path: `/logout`,
            method: 'POST',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.VoidApiResponse(response);
    }

    /**
     * Log out of Infra
     */
    async logout(initOverrides?: RequestInit): Promise<void> {
        await this.logoutRaw(initOverrides);
    }

}
