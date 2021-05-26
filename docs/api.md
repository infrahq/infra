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
* [Providers](#providers)
    * [Retrieve provider info](#retrieve-provider-info)

## Introduction

### Authentication

To authenticate with Infra, use tokens via the http `Authorization` header.

Using Bearer token auth:

```
curl https://infra.example.com/v1/users \
    -H "Authorization: Bearer Bearer eyJhbGci..._rY2PSRP5HA-g"
```

## Users

### Endpoints

```http
      POST /v1/users
       GET /v1/users
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
| `password`   | `string` | **Required** Password of user           |


#### Request

```bash
curl https://infra.example.com/v1/users \
    -H "Authorization: Bearer eyJhbGci..._rY2PSRP5HA-g" \
    -d email=user@example.com \
    -d password=passw0rd
```

#### Response

```json
{
    "id": "1",
    "email": "user@example.com",
    "created": 1620855986,
    "updated": 1620855986,
    "provider": "infra"
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
    -H "Authorization: Bearer eyJhbGci..._rY2PSRP5HA-g"
```

#### Response

```json
{
    "data": [
        {
            "id": "1",
            "email": "user@example.com",
            "created": 1620855986,
            "updated": 1620855986,
            "provider": "infra"
        },
    ]
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
    -H "Authorization: Bearer eyJhbGci..._rY2PSRP5HA-g" \
    -X DELETE
```

#### Response

```json
{
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
```

#### Parameters

| Parameter         | Type     | Description                                   |
| :--------         | :------- | :--------------------------------             |
| `code`            | `string` | Authorization code from provider (e.g. Okta)  |
| `email`           | `string` | User's email username                         |
| `password`        | `string` | User's password                               |


#### Request (Username & password)

```
curl https://infra.acme.com/v1/tokens \
    -d email=user@acme.com \
    -d password=passw0rd
```

#### Request (Exchange provider code for token)

```
curl https://infra.acme.com/v1/tokens \
    -d code=n9v7shdfv07shvps87hvp \
    -d provider=okta
```

#### Response

```json
{
    "token": "eyJhbGci..._rY2PSRP5HA-g"
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
curl https://infra.example.com/v1/providers
```

#### Response

```json
{
    "okta": {
        "client-id": "08jf308jesfdksnf9w3un",
        "domain": "example.okta.com"
    }
}
```