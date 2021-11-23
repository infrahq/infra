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

import { exists, mapValues } from '../runtime';
import {
    Role,
    RoleFromJSON,
    RoleFromJSONTyped,
    RoleToJSON,
    User,
    UserFromJSON,
    UserFromJSONTyped,
    UserToJSON,
} from './';

/**
 * 
 * @export
 * @interface Group
 */
export interface Group {
    /**
     * 
     * @type {string}
     * @memberof Group
     */
    id: string;
    /**
     * 
     * @type {string}
     * @memberof Group
     */
    name: string;
    /**
     * created time in seconds since 1970-01-01
     * @type {number}
     * @memberof Group
     */
    created: number;
    /**
     * updated time in seconds since 1970-01-01
     * @type {number}
     * @memberof Group
     */
    updated: number;
    /**
     * 
     * @type {string}
     * @memberof Group
     */
    providerID: string;
    /**
     * 
     * @type {Array<User>}
     * @memberof Group
     */
    users: Array<User>;
    /**
     * 
     * @type {Array<Role>}
     * @memberof Group
     */
    roles: Array<Role>;
}

export function GroupFromJSON(json: any): Group {
    return GroupFromJSONTyped(json, false);
}

export function GroupFromJSONTyped(json: any, ignoreDiscriminator: boolean): Group {
    if ((json === undefined) || (json === null)) {
        return json;
    }
    return {
        
        'id': json['id'],
        'name': json['name'],
        'created': json['created'],
        'updated': json['updated'],
        'providerID': json['providerID'],
        'users': ((json['users'] as Array<any>).map(UserFromJSON)),
        'roles': ((json['roles'] as Array<any>).map(RoleFromJSON)),
    };
}

export function GroupToJSON(value?: Group | null): any {
    if (value === undefined) {
        return undefined;
    }
    if (value === null) {
        return null;
    }
    return {
        
        'id': value.id,
        'name': value.name,
        'created': value.created,
        'updated': value.updated,
        'providerID': value.providerID,
        'users': ((value.users as Array<any>).map(UserToJSON)),
        'roles': ((value.roles as Array<any>).map(RoleToJSON)),
    };
}


