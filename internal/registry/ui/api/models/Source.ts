/**
 * Infra API
 * Infra REST API
 *
 * OpenAPI spec version: 0.1.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

import { SourceOkta } from './SourceOkta';
import { HttpFile } from '../http/http';

export class Source {
    'id': string;
    'created': number;
    'updated': number;
    'okta'?: SourceOkta;

    static readonly discriminator: string | undefined = undefined;

    static readonly attributeTypeMap: Array<{name: string, baseName: string, type: string, format: string}> = [
        {
            "name": "id",
            "baseName": "id",
            "type": "string",
            "format": ""
        },
        {
            "name": "created",
            "baseName": "created",
            "type": "number",
            "format": "int64"
        },
        {
            "name": "updated",
            "baseName": "updated",
            "type": "number",
            "format": "int64"
        },
        {
            "name": "okta",
            "baseName": "okta",
            "type": "SourceOkta",
            "format": ""
        }    ];

    static getAttributeTypeMap() {
        return Source.attributeTypeMap;
    }
    
    public constructor() {
    }
}

