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
    ApiService,
    ApiServiceFromJSON,
    ApiServiceToJSON,
    ApiServiceCreateRequest,
    ApiServiceCreateRequestFromJSON,
    ApiServiceCreateRequestToJSON,
    Service,
    ServiceFromJSON,
    ServiceToJSON,
} from '../models';

export interface CreateApiServiceRequest {
    body: ApiServiceCreateRequest;
}

/**
 * 
 */
export class ServicesApi extends runtime.BaseAPI {

    /**
     * Register an API service
     */
    async createApiServiceRaw(requestParameters: CreateApiServiceRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<ApiService>> {
        if (requestParameters.body === null || requestParameters.body === undefined) {
            throw new runtime.RequiredError('body','Required parameter requestParameters.body was null or undefined when calling createApiService.');
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
            path: `/services/apis`,
            method: 'POST',
            headers: headerParameters,
            query: queryParameters,
            body: ApiServiceCreateRequestToJSON(requestParameters.body),
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => ApiServiceFromJSON(jsonValue));
    }

    /**
     * Register an API service
     */
    async createApiService(requestParameters: CreateApiServiceRequest, initOverrides?: RequestInit): Promise<ApiService> {
        const response = await this.createApiServiceRaw(requestParameters, initOverrides);
        return await response.value();
    }

    /**
     * List services
     */
    async listServicesRaw(initOverrides?: RequestInit): Promise<runtime.ApiResponse<Array<Service>>> {
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
            path: `/services`,
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => jsonValue.map(ServiceFromJSON));
    }

    /**
     * List services
     */
    async listServices(initOverrides?: RequestInit): Promise<Array<Service>> {
        const response = await this.listServicesRaw(initOverrides);
        return await response.value();
    }

}
