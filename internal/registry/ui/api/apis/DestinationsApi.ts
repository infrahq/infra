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
    Destination,
    DestinationFromJSON,
    DestinationToJSON,
    DestinationCreateRequest,
    DestinationCreateRequestFromJSON,
    DestinationCreateRequestToJSON,
} from '../models';

export interface CreateDestinationRequest {
    body: DestinationCreateRequest;
}

export interface GetDestinationRequest {
    id: string;
}

export interface ListDestinationsRequest {
    name?: string;
    type?: string;
}

/**
 * 
 */
export class DestinationsApi extends runtime.BaseAPI {

    /**
     * Create a destination
     */
    async createDestinationRaw(requestParameters: CreateDestinationRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<Destination>> {
        if (requestParameters.body === null || requestParameters.body === undefined) {
            throw new runtime.RequiredError('body','Required parameter requestParameters.body was null or undefined when calling createDestination.');
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
            path: `/destinations`,
            method: 'POST',
            headers: headerParameters,
            query: queryParameters,
            body: DestinationCreateRequestToJSON(requestParameters.body),
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => DestinationFromJSON(jsonValue));
    }

    /**
     * Create a destination
     */
    async createDestination(requestParameters: CreateDestinationRequest, initOverrides?: RequestInit): Promise<Destination> {
        const response = await this.createDestinationRaw(requestParameters, initOverrides);
        return await response.value();
    }

    /**
     * Get destination by ID
     */
    async getDestinationRaw(requestParameters: GetDestinationRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<Destination>> {
        if (requestParameters.id === null || requestParameters.id === undefined) {
            throw new runtime.RequiredError('id','Required parameter requestParameters.id was null or undefined when calling getDestination.');
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
            path: `/destinations/{id}`.replace(`{${"id"}}`, encodeURIComponent(String(requestParameters.id))),
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => DestinationFromJSON(jsonValue));
    }

    /**
     * Get destination by ID
     */
    async getDestination(requestParameters: GetDestinationRequest, initOverrides?: RequestInit): Promise<Destination> {
        const response = await this.getDestinationRaw(requestParameters, initOverrides);
        return await response.value();
    }

    /**
     * List destinations
     */
    async listDestinationsRaw(requestParameters: ListDestinationsRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<Array<Destination>>> {
        const queryParameters: any = {};

        if (requestParameters.name !== undefined) {
            queryParameters['name'] = requestParameters.name;
        }

        if (requestParameters.type !== undefined) {
            queryParameters['type'] = requestParameters.type;
        }

        const headerParameters: runtime.HTTPHeaders = {};

        if (this.configuration && this.configuration.accessToken) {
            const token = this.configuration.accessToken;
            const tokenString = await token("bearerAuth", []);

            if (tokenString) {
                headerParameters["Authorization"] = `Bearer ${tokenString}`;
            }
        }
        const response = await this.request({
            path: `/destinations`,
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => jsonValue.map(DestinationFromJSON));
    }

    /**
     * List destinations
     */
    async listDestinations(requestParameters: ListDestinationsRequest, initOverrides?: RequestInit): Promise<Array<Destination>> {
        const response = await this.listDestinationsRaw(requestParameters, initOverrides);
        return await response.value();
    }

}
