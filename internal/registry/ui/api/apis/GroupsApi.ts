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
    Group,
    GroupFromJSON,
    GroupToJSON,
} from '../models';

/**
 * 
 */
export class GroupsApi extends runtime.BaseAPI {

    /**
     * List groups
     */
    async listGroupsRaw(initOverrides?: RequestInit): Promise<runtime.ApiResponse<Array<Group>>> {
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
            path: `/groups`,
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => jsonValue.map(GroupFromJSON));
    }

    /**
     * List groups
     */
    async listGroups(initOverrides?: RequestInit): Promise<Array<Group>> {
        const response = await this.listGroupsRaw(initOverrides);
        return await response.value();
    }

}
