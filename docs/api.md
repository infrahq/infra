# API Reference

* [Introduction](#introduction)
    * [Authentication](#authentication)
* [Users](#users)
    * [Create a user](#create-a-user)
    * [List all users](#list-all-users)
    * [Retrieve a user](#retrieve-a-user)
    * [Delete a user](#delete-a-user)
* [Tokens](#tokens)
    * [Create a token](#create-a-token)
    * [List all tokens](#list-all-tokens)
    * [Delete a token](#delete-a-token)
* [Providers](#providers)
    * [Retrieve provider info](#retrieve-provider-info)

## Introduction

### Authentication

To authenticate with Infra Engine, use tokens. Tokens can be provided in two ways:
* `username` with Basic Auth
* `Bearer` token auth

For example, using Basic auth:

```
curl https://infra.acme.com/v1/users \
    -u sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi:
```

Using Bearer token auth:

```
curl https://infra.acme.com/v1/users \
    -H "Authorization: Bearer sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi"
```

## Users

### Endpoints

```http
      POST /v1/users
       GET /v1/users
       GET /v1/users/:id
    DELETE /v1/users/:id
```

### Create a user

```http
    POST /v1/users
```

#### Parameters

| Parameter    | Type     | Description                             |
| :--------    | :------- | :-------------------------------------- |
| `email`      | `string` | **Required** Email of user              |


#### Request

```bash
curl https://infra.acme.com/v1/users \
    -u sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi: \
    -d email=test@acme.com
```

#### Response

```json
{
    "id": "usr_LB4MsQycuLEH",
    "email": "test@infrahq.com",
    "created": 1620855986,
    "providers": ["okta"]
}
```

### List all users

```http
    GET /v1/users
```

#### Parameters

No parameters

#### Request

```
curl https://infra.acme.com/v1/users \
    -u sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi:
```

#### Response

```json
{
    "data": [
        {
            "id": "usr_mvm8YVTvOGY4",
            "email": "tom@acme.com",
            "created": 1620845768,
            "providers": ["okta"]
        },
    ],
    "url":"/v1/users"
}
```


### Retrieve a user

```http
    GET /v1/users/:id
```

#### Parameters

No parameters.

#### Request

```
curl https://infra.acme.com/v1/users/usr_LB4MsQycuLEH \
    -u sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi:
```

#### Response

```json
{
    "id": "usr_LB4MsQycuLEH",
    "email": "test@infrahq.com",
    "created": 1620855986,
    "providers": ["okta"],
    "permission": "view"
}
```

### Delete a user

```http
    DELETE /v1/users/:id
```

#### Parameters

No parameters.

#### Request

```
curl https://infra.acme.com/v1/users/usr_LB4MsQycuLEH \
    -u sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi: \
    -X DELETE
```

#### Response

```json
{
    "id":"usr_LB4MsQycuLEH",
    "deleted": true,
}
```

## Tokens

### Endpoints

```http
    POST /v1/tokens
```

### Create a token

```http
    POST /v1/tokens
     GET /v1/tokens
  DELETE /v1/tokens
```

#### Parameters

| Parameter         | Type     | Description                       |
| :--------         | :------- | :-------------------------------- |
| `code`            | `string` | Authorization code from provider  |
| `code_provider`   | `string` | Provider to verify code           |
| `user`            | `string` | User for which to generate token  |

#### Request (Rotate a token)

```
curl https://infra.acme.com/v1/tokens \
    -u sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi:
```

#### Request (Create token for another user)

```
curl https://infra.acme.com/v1/tokens \
    -u sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi:
    -d user=usr_LB4MsQycuLEH
```

#### Request (Exchange provider code for token)

```
curl https://infra.acme.com/v1/tokens \
    -d code=n9v7shdfv07shvps87hvpse8hspe8ch
    -d provider=okta
```

#### Response

```json
{
    "token": {
        "id": "tk_1xeLhL379bJO",
        "user": "usr_LB4MsQycuLEH",
        "expires": 1620861101,
        "created": 1620857501,
    },
    "secret_token": "sk_1xeLhL379bJOkytFbDizJixDrcqbbbnZM78gI7HLjqJp"
}
```

### List all tokens

```http
    GET /v1/tokens
```

#### Parameters

No parameters

#### Request

```
curl https://infra.acme.com/v1/tokens \
    -u sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi:
```

#### Response

```json
{
    "data": [
        {
            "id": "tk_1xeLhL379bJO",
            "expires": 1620861101,
            "created": 1620857501,
            "user": {
                "id": "usr_mvm8YVTvOGY4",
                "email": "tom@acme.com",
                "created": 1620845768,
                "providers": ["okta"],
                "permission": "view",
            },
        },
    ],
    "url":"/v1/tokens"
}
```

### Delete a token

```http
    DELETE /v1/tokens/:id
```

#### Parameters

No parameters.

#### Request

```
curl https://infra.acme.com/v1/tokens/tk_1xeLhL379bJO \
    -u sk_mnrdvosho472npiwdnakjsdn9as74sdo1dfi: \
    -X DELETE
```

#### Response

```json
{
    "id":"tk_1xeLhL379bJO",
    "deleted": true,
}
```

## Providers

```http
  GET /v1/providers
```

### Retrieve provider information

```http
  GET /v1/providers
```

#### Parameters

No parameters

#### Request

```
curl https://infra.acme.com/v1/providers
```

#### Response

```json
{
    "okta": {
        "client-id": "08jf308jesfdksnf9w3un",
        "domain": "okta.acme.com"
    }
}
```