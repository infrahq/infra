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
    Grant,
    GrantFromJSON,
    GrantToJSON,
    GrantRequest,
    GrantRequestFromJSON,
    GrantRequestToJSON,
} from '../models';

export interface CreateGrantRequest {
    body: GrantRequest;
}

export interface DeleteGrantRequest {
    id: string;
}

export interface GetGrantRequest {
    id: string;
}

export interface UpdateGrantRequest {
    id: string;
    grantRequest: GrantRequest;
}

/**
 * 
 */
export class GrantsApi extends runtime.BaseAPI {

    /**
     * Create a grant
     */
    async createGrantRaw(requestParameters: CreateGrantRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<Grant>> {
        if (requestParameters.body === null || requestParameters.body === undefined) {
            throw new runtime.RequiredError('body','Required parameter requestParameters.body was null or undefined when calling createGrant.');
        }

        const queryParameters: any = {};

        const headerParameters: runtime.HTTPHeaders = {};

        headerParameters['Content-Type'] = 'application/json';

        if (this.configuration && this.configuration.accessToken) {
            const token = this.configuration.accessToken;
            const tokenString = await token("bearerAuth", []);

            if (tokenString) {
                headerParameters["Authorization"] = `Bearer ${tokenString}`;
            }
        }
        const response = await this.request({
            path: `/grants`,
            method: 'POST',
            headers: headerParameters,
            query: queryParameters,
            body: GrantRequestToJSON(requestParameters.body),
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => GrantFromJSON(jsonValue));
    }

    /**
     * Create a grant
     */
    async createGrant(requestParameters: CreateGrantRequest, initOverrides?: RequestInit): Promise<Grant> {
        const response = await this.createGrantRaw(requestParameters, initOverrides);
        return await response.value();
    }

    /**
     * Delete a grant by ID
     */
    async deleteGrantRaw(requestParameters: DeleteGrantRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<void>> {
        if (requestParameters.id === null || requestParameters.id === undefined) {
            throw new runtime.RequiredError('id','Required parameter requestParameters.id was null or undefined when calling deleteGrant.');
        }

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
            path: `/grants/{id}`.replace(`{${"id"}}`, encodeURIComponent(String(requestParameters.id))),
            method: 'DELETE',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.VoidApiResponse(response);
    }

    /**
     * Delete a grant by ID
     */
    async deleteGrant(requestParameters: DeleteGrantRequest, initOverrides?: RequestInit): Promise<void> {
        await this.deleteGrantRaw(requestParameters, initOverrides);
    }

    /**
     * Get grant
     */
    async getGrantRaw(requestParameters: GetGrantRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<Grant>> {
        if (requestParameters.id === null || requestParameters.id === undefined) {
            throw new runtime.RequiredError('id','Required parameter requestParameters.id was null or undefined when calling getGrant.');
        }

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
            path: `/grants/{id}`.replace(`{${"id"}}`, encodeURIComponent(String(requestParameters.id))),
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => GrantFromJSON(jsonValue));
    }

    /**
     * Get grant
     */
    async getGrant(requestParameters: GetGrantRequest, initOverrides?: RequestInit): Promise<Grant> {
        const response = await this.getGrantRaw(requestParameters, initOverrides);
        return await response.value();
    }

    /**
     * List grants
     */
    async listGrantsRaw(initOverrides?: RequestInit): Promise<runtime.ApiResponse<Array<Grant>>> {
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
            path: `/grants`,
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => jsonValue.map(GrantFromJSON));
    }

    /**
     * List grants
     */
    async listGrants(initOverrides?: RequestInit): Promise<Array<Grant>> {
        const response = await this.listGrantsRaw(initOverrides);
        return await response.value();
    }

    /**
     * Update a grant by ID
     */
    async updateGrantRaw(requestParameters: UpdateGrantRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<Grant>> {
        if (requestParameters.id === null || requestParameters.id === undefined) {
            throw new runtime.RequiredError('id','Required parameter requestParameters.id was null or undefined when calling updateGrant.');
        }

        if (requestParameters.grantRequest === null || requestParameters.grantRequest === undefined) {
            throw new runtime.RequiredError('grantRequest','Required parameter requestParameters.grantRequest was null or undefined when calling updateGrant.');
        }

        const queryParameters: any = {};

        const headerParameters: runtime.HTTPHeaders = {};

        headerParameters['Content-Type'] = 'application/json';

        const response = await this.request({
            path: `/grants/{id}`.replace(`{${"id"}}`, encodeURIComponent(String(requestParameters.id))),
            method: 'PUT',
            headers: headerParameters,
            query: queryParameters,
            body: GrantRequestToJSON(requestParameters.grantRequest),
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => GrantFromJSON(jsonValue));
    }

    /**
     * Update a grant by ID
     */
    async updateGrant(requestParameters: UpdateGrantRequest, initOverrides?: RequestInit): Promise<Grant> {
        const response = await this.updateGrantRaw(requestParameters, initOverrides);
        return await response.value();
    }

}
