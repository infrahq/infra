# API Reference

## Overview

### Authentication

Infra authenticates access to the API via **Access Keys**. Access keys are tied to a specific organization. Access keys can be created and managed in the Infra Dashboard. Be sure to securely store access keys: they have the same level of permissions as the user who creates them.

### Versioning

**The current API version is** `0.18.1`

The Infra API is versioned. Requests to the API must contain a header named `Infra-Version`. Set this header to the version matching the API docs reference you're using, or the version of the server you're using. Once you set this value you can forget about it until you want to use features from newer API versions. A valid version header looks like this:

```
Infra-Version: 0.18.1
```

### Pagination

Every **list** response in the API is paginated (split into pages). If the page number and limit (page size) aren't specified, then the response will contain the first page of 100 records.

#### Paginated requests

List requests accept additional query parameters:

| Parameter | Type | Description                                         |
| --------- | ---- | --------------------------------------------------- |
| page      | int  | Page number to retrieve                             |
| limit     | int  | Number of objects to retrieve per page (up to 1000) |

Examples:

- `GET /api/grants?page=2` returns the second page of 100 grants.
- `GET /api/users?page=1&limit=10` returns the first page of 10 users
- `GET /api/users?page=2&limit=10` returns the second page of 10 users

#### Paginated responses

List responses take the form:

```json
{
  "page": 4,
  "limit": 10,
  "totalPages": 40,
  "totalCount": 395
}
```

Use the `totalPages` field to determine the number of pages needed to request to get all records with the given limit.

### Dates

Infra uses the `RFC 3339` timestamp format for any date fields.

### Long polling

A select few list endpoints support **long polling**. Long polling allows for near-instant updates by responding only when data has changed. To make a long polling request, include the `lastUpdateIndex` query parameter. Use `1` for the initial request.

```bash
GET /api/grants?lastUpdateIndex=1
```

Long polling responses include an additional header, `Last-Update-Index` with the last index to be used in subsequent long-polling requests. Include the value of this header in subsequent requests:

```
GET /api/grants?lastUpdateIndex=102850182
```

### Errors

Errors in Infra's API all follow a consistent format:

```json
{
  "code": 400,
  "message": "Name contains invalid characters",
  "fieldErrors": [
    {
      "name": ["invalid character at position 4"]
    }
  ]
}
```

#### Status Codes

| Status Code             | Summary                                                                 |
| ----------------------- | ----------------------------------------------------------------------- |
| 200                     | The request worked as expected                                          |
| 201 - Created           | The requested resource was created                                      |
| 400 - Bad request       | Invalid request parameters                                              |
| 401 - Unauthorized      | No access key was provided                                              |
| 403 - Unauthorized      | This user or access key does not have permission to perform the request |
| 404 - Not found         | The requested resource was not found                                    |
| 409 - Conflict          | The request conflicts with an existing resource (e.g. a duplicate)      |
| 429 - Too many requests | Too many requests have been sent to the API                             |
| 500 - Server error      | There was an internal error with Infra's API                            |

## Access Grants

**Access grants** are the core resource in Infra that decides access control. They tie together three concepts:

1. The user or group
2. The privilege (e.g. a role or permission)
3. The resource (e.g. a server, cluster or namespace)

### List access

```
GET /api/grants
```

This endpoint offers a way to list all grants on the system, and optionally filter using any of the query parameters. A request can include no query parameters, one query parameter, or many to help you find the grants that meet any criteria. Also supports long polling using the lastUpdateIndex query parameter. See [Long Polling](#long-polling) for more information.

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Query Parameters

| Field           | Format  | Description                                                                                                 |
| --------------- | ------- | ----------------------------------------------------------------------------------------------------------- |
| user            | string  | ID of user granted access                                                                                   |
| group           | string  | ID of group granted access                                                                                  |
| resource        | string  | a resource name                                                                                             |
| destination     | string  | name of the destination where a connector is installed                                                      |
| privilege       | string  | a role or permission                                                                                        |
| showInherited   | boolean | if true, this field includes grants that the user inherits through groups                                   |
| showSystem      | boolean | if true, this shows the connector and other internal grants                                                 |
| lastUpdateIndex | integer | set this to the value of the Last-Update-Index response header to block until the list results have changed |
| page            | integer | Page number to retrieve                                                                                     |
| limit           | integer | Number of objects to retrieve per page (up to 1000)                                                         |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/grants?user=6TjWTAgYYu \
    &group=6k3Eqcqu6B \
    &resource=production.namespace \
    &destination=production \
    &privilege=view \
    &showInherited=true \
    &showSystem=false \
    &page=1 \
    &limit=100 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns list of all grants matching the criteria specified in the query parameters

#### Example response

```json
{
  "count": "100",
  "items": [
    {
      "created": "2022-03-14T09:48:00Z",
      "createdBy": "4yJ3n3D8E2",
      "group": "3zMaadcd2U",
      "id": "3w9XyTrkzk",
      "privilege": "admin",
      "resource": "production.namespace",
      "updated": "2022-03-14T09:48:00Z",
      "user": "6hNnjfjVcc"
    }
  ],
  "limit": 100,
  "page": 1,
  "totalCount": 485,
  "totalPages": 5
}
```

#### Example response parameters

| Field           | Type   | Description                                                |
| --------------- | ------ | ---------------------------------------------------------- |
| items.created   | string | formatted as an RFC3339 date-time                          |
| items.createdBy | string | id of the user that created the grant                      |
| items.group     | string | GroupID for a group being granted access                   |
| items.id        | string | ID of grant created                                        |
| items.privilege | string | a role or permission                                       |
| items.resource  | string | a resource name in Infra&#39;s Universal Resource Notation |
| items.updated   | string | formatted as an RFC3339 date-time                          |
| items.user      | string | UserID for a user being granted access                     |

### Grant access

```
POST /api/grants
```

This endpoint will allow you to create a new grant allowing a user or group a specified level of access to any resource.

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Body Parameters

| Field     | Format | Description                                                |
| --------- | ------ | ---------------------------------------------------------- |
| group     | string | ID of the group granted access                             |
| groupName | string | Name of the group granted access                           |
| privilege | string | a role or permission                                       |
| resource  | string | a resource name in Infra&#39;s Universal Resource Notation |
| user      | string | ID of the user granted access                              |
| userName  | string | Name of the user granted access                            |

#### Example Request

```shell
curl -X POST https://api.infrahq.com/api/grants \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY' \
  -d '{
    "privilege": "view",
    "resource": "production",
    "user": "6kdoMDd6PA",
  }'
```

#### Returns

Returns the grant object, with an additional field `wasCreated` that is `true` if this grant did not previously exist.

#### Example response

```json
{
  "created": "2022-03-14T09:48:00Z",
  "createdBy": "4yJ3n3D8E2",
  "group": "3zMaadcd2U",
  "id": "3w9XyTrkzk",
  "privilege": "admin",
  "resource": "production.namespace",
  "updated": "2022-03-14T09:48:00Z",
  "user": "6hNnjfjVcc",
  "wasCreated": true
}
```

#### Example response parameters

| Field      | Type    | Description                                                                        |
| ---------- | ------- | ---------------------------------------------------------------------------------- |
| created    | string  | formatted as an RFC3339 date-time                                                  |
| createdBy  | string  | id of the user that created the grant                                              |
| group      | string  | GroupID for a group being granted access                                           |
| id         | string  | ID of grant created                                                                |
| privilege  | string  | a role or permission                                                               |
| resource   | string  | a resource name in Infra&#39;s Universal Resource Notation                         |
| updated    | string  | formatted as an RFC3339 date-time                                                  |
| user       | string  | UserID for a user being granted access                                             |
| wasCreated | boolean | Indicates that grant was successfully created, false it already existed beforehand |

### Update access

```
PATCH /api/grants
```

Allows for bulk adding and removing grants

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Body Parameters

| Field          | Format | Description                                         |
| -------------- | ------ | --------------------------------------------------- |
| grantsToAdd    | array  | List of grant objects. See POST api/grants for more |
| grantsToRemove | array  | List of grant objects. See POST api/grants for more |

#### Example Request

```shell
curl -X PATCH https://api.infrahq.com/api/grants \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY' \
  -d '{
    "grantsToAdd": [{
      "userName": "bob@example.com",
      "privilege": "view",
      "resource": "aws-dev"
    },
    {
      "userName": "cindy@example.com",
      "privilege": "admin",
      "resource": "aws-dev"
    }]
  }'
```

#### Returns

Returns a response code with no body

#### Example response

```json
Empty Response
```

#### Example response parameters

Empty Response

### List a specific access grant

```
GET /api/grants/{id}
```

Searches for a specific access grant by ID

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Path Parameters

| Field | Format | Description                 |
| ----- | ------ | --------------------------- |
| id    | string | ID of the grant to retrieve |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/grants/4yJ3n3D8E2 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns all the details for a grant with the ID specified in the path

#### Example response

```json
{
  "created": "2022-03-14T09:48:00Z",
  "createdBy": "4yJ3n3D8E2",
  "group": "3zMaadcd2U",
  "id": "3w9XyTrkzk",
  "privilege": "admin",
  "resource": "production.namespace",
  "updated": "2022-03-14T09:48:00Z",
  "user": "6hNnjfjVcc"
}
```

#### Example response parameters

| Field     | Type   | Description                                                |
| --------- | ------ | ---------------------------------------------------------- |
| created   | string | formatted as an RFC3339 date-time                          |
| createdBy | string | id of the user that created the grant                      |
| group     | string | GroupID for a group being granted access                   |
| id        | string | ID of grant created                                        |
| privilege | string | a role or permission                                       |
| resource  | string | a resource name in Infra&#39;s Universal Resource Notation |
| updated   | string | formatted as an RFC3339 date-time                          |
| user      | string | UserID for a user being granted access                     |

### Remove access

```
DELETE /api/grants/{id}
```

Deletes any grant with the specified id

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Path Parameters

| Field | Format | Description               |
| ----- | ------ | ------------------------- |
| id    | string | ID of the grant to remove |

#### Example Request

```shell
curl -X DELETE https://api.infrahq.com/api/grants/4yJ3n3D8E2 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns an empty body with a response code

#### Example response

```json
Empty Response
```

#### Example response parameters

Empty Response

## Managing Users

**Users** represent the humans that would connect to a cluster and are defined by an email address.

### List users

```
GET /api/users
```

List all the users that match an optional query

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Query Parameters

| Field      | Format  | Description                                                |
| ---------- | ------- | ---------------------------------------------------------- |
| name       | string  | Name of the user                                           |
| group      | string  | Group the user belongs to                                  |
| ids        | array   | List of User IDs                                           |
| showSystem | boolean | if true, this shows the connector and other internal users |
| page       | integer | Page number to retrieve                                    |
| limit      | integer | Number of objects to retrieve per page (up to 1000)        |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/users?name=bob@example.com \
    &group=admins \
    &showSystem=false \
    &page=1 \
    &limit=100 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns an array of user objects that match an optional query

#### Example response

```json
{
  "count": "100",
  "items": [
    {
      "created": "2022-03-14T09:48:00Z",
      "id": "4yJ3n3D8E2",
      "lastSeenAt": "2022-03-14T09:48:00Z",
      "name": "bob@example.com",
      "providerNames": ["okta"],
      "updated": "2022-03-14T09:48:00Z"
    }
  ],
  "limit": 100,
  "page": 1,
  "totalCount": 485,
  "totalPages": 5
}
```

#### Example response parameters

| Field               | Type   | Description                            |
| ------------------- | ------ | -------------------------------------- |
| items.created       | string | formatted as an RFC3339 date-time      |
| items.id            | string | User ID                                |
| items.lastSeenAt    | string | formatted as an RFC3339 date-time      |
| items.name          | string | Name of the user                       |
| items.providerNames | array  | List of providers this user belongs to |
| items.updated       | string | formatted as an RFC3339 date-time      |

### Create users

```
POST /api/users
```

Create a user with a specified name. The next step after creation will depend on whether the server is hosted with Infra Cloud or self-hosted. If using Infra Cloud, the new user will receive an email inviting them to the server. If self-hosted, this API returns a one-time password which will need to be relayed to the user.

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Body Parameters

| Field | Format | Description                   |
| ----- | ------ | ----------------------------- |
| name  | string | Email address of the new user |

#### Example Request

```shell
curl -X POST https://api.infrahq.com/api/users \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY' \
  -d '{
    "name": "bob@example.com"
  }'
```

#### Returns

If using Infra (&lt;yourorg&gt;.infrahq.com), the response will show a UserID. If self-hosted, the response will show the ID and a one-time password.

#### Example response

```json
{
  "id": "4yJ3n3D8E2",
  "name": "bob@example.com",
  "oneTimePassword": "password"
}
```

#### Example response parameters

| Field           | Type   | Description                                        |
| --------------- | ------ | -------------------------------------------------- |
| id              | string | User ID                                            |
| name            | string | Email address of the user                          |
| oneTimePassword | string | One-time password (only returned when self-hosted) |

### Get a user

```
GET /api/users/{id}
```

Get a user with the specified ID

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Path Parameters

| Field | Format | Description                |
| ----- | ------ | -------------------------- |
| id    | string | ID of the user to retrieve |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/users/4yJ3n3D8E2 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns a single user object

#### Example response

```json
{
  "created": "2022-03-14T09:48:00Z",
  "id": "4yJ3n3D8E2",
  "lastSeenAt": "2022-03-14T09:48:00Z",
  "name": "bob@example.com",
  "providerNames": ["okta"],
  "updated": "2022-03-14T09:48:00Z"
}
```

#### Example response parameters

| Field         | Type   | Description                            |
| ------------- | ------ | -------------------------------------- |
| created       | string | formatted as an RFC3339 date-time      |
| id            | string | User ID                                |
| lastSeenAt    | string | formatted as an RFC3339 date-time      |
| name          | string | Name of the user                       |
| providerNames | array  | List of providers this user belongs to |
| updated       | string | formatted as an RFC3339 date-time      |

### Update a user password

```
PUT /api/users/{id}
```

Update a user&#39;s password. If the access key used to access this API belongs to an Infra Admin, then the old password does not need to be provided. Otherwise the old password is required. The password parameter is the new one-time password for the user.

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Path Parameters

| Field | Format | Description              |
| ----- | ------ | ------------------------ |
| id    | string | ID of the user to update |

#### Body Parameters

| Field       | Format | Description                                                                                      |
| ----------- | ------ | ------------------------------------------------------------------------------------------------ |
| oldPassword | string | Old password for the user. Only required when the access key used is not owned by an Infra admin |
| password    | string | New one-time password for the user                                                               |

#### Example Request

```shell
curl -X PUT https://api.infrahq.com/api/users/4yJ3n3D8E2 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY' \
  -d '{
    "oldPassword": "oldpassword",
    "password": "newpassword"
  }'
```

#### Returns

Returns a single user object

#### Example response

```json
{
  "created": "2022-03-14T09:48:00Z",
  "id": "4yJ3n3D8E2",
  "lastSeenAt": "2022-03-14T09:48:00Z",
  "name": "bob@example.com",
  "providerNames": ["okta"],
  "updated": "2022-03-14T09:48:00Z"
}
```

#### Example response parameters

| Field         | Type   | Description                            |
| ------------- | ------ | -------------------------------------- |
| created       | string | formatted as an RFC3339 date-time      |
| id            | string | User ID                                |
| lastSeenAt    | string | formatted as an RFC3339 date-time      |
| name          | string | Name of the user                       |
| providerNames | array  | List of providers this user belongs to |
| updated       | string | formatted as an RFC3339 date-time      |

### Delete a user

```
DELETE /api/users/{id}
```

Delete the user by User ID

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Path Parameters

| Field | Format | Description              |
| ----- | ------ | ------------------------ |
| id    | string | ID of the user to remove |

#### Example Request

```shell
curl -X DELETE https://api.infrahq.com/api/users/4yJ3n3D8E2 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns an empty object

#### Example response

```json
Empty Response
```

#### Example response parameters

Empty Response

## Group Management

**Groups** are used in Infra to manage collections of users. A group can then be associated with a role and cluster via a grant and all users with the group will gain that role and and corresponding access to the cluster.

### List groups

```
GET /api/groups
```

List all the groups that match an optional query

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Query Parameters

| Field  | Format  | Description                                         |
| ------ | ------- | --------------------------------------------------- |
| name   | string  | Name of the group to retrieve                       |
| userID | string  | UserID of a user who is a member of the group       |
| page   | integer | Page number to retrieve                             |
| limit  | integer | Number of objects to retrieve per page (up to 1000) |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/groups?name=admins \
    &userID=4yJ3n3D8E2 \
    &page=1 \
    &limit=100 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns an array of objects describing each group that match the query

#### Example response

```json
{
  "count": "100",
  "items": [
    {
      "created": "2022-03-14T09:48:00Z",
      "id": "4yJ3n3D8E2",
      "name": "admins",
      "totalUsers": 14,
      "updated": "2022-03-14T09:48:00Z"
    }
  ],
  "limit": 100,
  "page": 1,
  "totalCount": 485,
  "totalPages": 5
}
```

#### Example response parameters

| Field            | Type    | Description                        |
| ---------------- | ------- | ---------------------------------- |
| items.created    | string  | formatted as an RFC3339 date-time  |
| items.id         | string  | Group ID                           |
| items.name       | string  | Name of the group                  |
| items.totalUsers | integer | Total number of users in the group |
| items.updated    | string  | formatted as an RFC3339 date-time  |

### Create a group

```
POST /api/groups
```

Create a new group with a specified name

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Body Parameters

| Field | Format | Description       |
| ----- | ------ | ----------------- |
| name  | string | Name of the group |

#### Example Request

```shell
curl -X POST https://api.infrahq.com/api/groups \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY' \
  -d '{
    "name": "development"
  }'
```

#### Returns

Returns the name and id of the new group

#### Example response

```json
{
  "created": "2022-03-14T09:48:00Z",
  "id": "4yJ3n3D8E2",
  "name": "admins",
  "totalUsers": 14,
  "updated": "2022-03-14T09:48:00Z"
}
```

#### Example response parameters

| Field      | Type    | Description                        |
| ---------- | ------- | ---------------------------------- |
| created    | string  | formatted as an RFC3339 date-time  |
| id         | string  | Group ID                           |
| name       | string  | Name of the group                  |
| totalUsers | integer | Total number of users in the group |
| updated    | string  | formatted as an RFC3339 date-time  |

### Get a group by ID

```
GET /api/groups/{id}
```

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Path Parameters

| Field | Format | Description                 |
| ----- | ------ | --------------------------- |
| id    | string | ID of the group to retrieve |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/groups/4yJ3n3D8E2 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

#### Example response

```json
{
  "created": "2022-03-14T09:48:00Z",
  "id": "4yJ3n3D8E2",
  "name": "admins",
  "totalUsers": 14,
  "updated": "2022-03-14T09:48:00Z"
}
```

#### Example response parameters

| Field      | Type    | Description                        |
| ---------- | ------- | ---------------------------------- |
| created    | string  | formatted as an RFC3339 date-time  |
| id         | string  | Group ID                           |
| name       | string  | Name of the group                  |
| totalUsers | integer | Total number of users in the group |
| updated    | string  | formatted as an RFC3339 date-time  |

### Delete a group

```
DELETE /api/groups/{id}
```

Delete a group with the specified ID. You can find the ID of the group using either `GET api/groups` or `GET api/groups/{id}`.

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Path Parameters

| Field | Format | Description               |
| ----- | ------ | ------------------------- |
| id    | string | ID of the group to remove |

#### Example Request

```shell
curl -X DELETE https://api.infrahq.com/api/groups/4yJ3n3D8E2 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns an empty response

#### Example response

```json
Empty Response
```

#### Example response parameters

Empty Response

## Managing Providers

**Providers** is short for OIDC providers and, when used in Infra, are the authoritative source of information about users and groups.

### List providers

```
GET /api/providers
```

List all the providers that match an optional query

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |

#### Query Parameters

| Field | Format  | Description                                         |
| ----- | ------- | --------------------------------------------------- |
| name  | string  | Name of the provider                                |
| page  | integer | Page number to retrieve                             |
| limit | integer | Number of objects to retrieve per page (up to 1000) |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/providers?name=okta \
    &page=1 \
    &limit=100 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1'
```

#### Returns

Returns an array of provider objects that match an optional query

#### Example response

```json
{
  "count": "100",
  "items": [
    {
      "authURL": "https://example.com/oauth2/v1/authorize",
      "clientID": "0oapn0qwiQPiMIyR35d6",
      "created": "2022-03-14T09:48:00Z",
      "id": "4yJ3n3D8E2",
      "kind": "oidc",
      "name": "okta",
      "scopes": "['openid', 'email']",
      "updated": "2022-03-14T09:48:00Z",
      "url": "infrahq.okta.com"
    }
  ],
  "limit": 100,
  "page": 1,
  "totalCount": 485,
  "totalPages": 5
}
```

#### Example response parameters

| Field          | Type   | Description                                   |
| -------------- | ------ | --------------------------------------------- |
| items.authURL  | string | Authorize endpoint for the OIDC provider      |
| items.clientID | string | Client ID for the OIDC provider               |
| items.created  | string | formatted as an RFC3339 date-time             |
| items.id       | string | Provider ID                                   |
| items.kind     | string | Kind of provider                              |
| items.name     | string | Name of the provider                          |
| items.scopes   | array  | Scopes set in the OIDC provider configuration |
| items.updated  | string | formatted as an RFC3339 date-time             |
| items.url      | string | URL of the Infra Server                       |

## Working with Destinations

**Destinations** are where the connectors are installed to. An example of a destination would be a Kubernetes cluster.

### List all the destinations

```
GET /api/destinations
```

List all the destinations that match an optional query

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Query Parameters

| Field     | Format  | Description                                            |
| --------- | ------- | ------------------------------------------------------ |
| name      | string  | Name of the destination                                |
| kind      | string  | Kind of destination. eg. kubernetes or ssh or postgres |
| unique_id | string  | Unique ID generated by the connector                   |
| page      | integer | Page number to retrieve                                |
| limit     | integer | Number of objects to retrieve per page (up to 1000)    |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/destinations?name=production-cluster \
    &kind=kubernetes \
    &unique_id=94c2c570a20311180ec325fd56 \
    &page=1 \
    &limit=100 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns a paginated list of destinations

#### Example response

```json
{
  "count": "100",
  "items": [
    {
      "connected": true,
      "connection": {
        url: aa60eexample.us-west-2.elb.amazonaws.com,
        ca: -----BEGIN CERTIFICATE-----
            MIIDNTCCAh2gAwIBAgIRALRetnpcTo9O3V2fAK3ix+c
            -----END CERTIFICATE-----
      },
      "created": "2022-03-14T09:48:00Z",
      "id": "7a1b26b33F",
      "kind": "kubernetes",
      "lastSeen": "2022-03-14T09:48:00Z",
      "name": "production-cluster",
      "resources": ['default', 'kube-system'],
      "roles": ['cluster-admin', 'admin', 'edit', 'view', 'exec', 'logs', 'port-forward'],
      "uniqueID": "94c2c570a20311180ec325fd56",
      "updated": "2022-03-14T09:48:00Z",
      "version": "0.18.1",
    }
  ],
  "limit": 100,
  "page": 1,
  "totalCount": 485,
  "totalPages": 5
}

```

#### Example response parameters

| Field            | Type    | Description                                                                                     |
| ---------------- | ------- | ----------------------------------------------------------------------------------------------- |
| items.connected  | boolean | Shows if the destination is currently connected                                                 |
| items.connection | object  | Object that includes the URL and CA for the destination                                         |
| items.created    | string  | formatted as an RFC3339 date-time                                                               |
| items.id         | string  | ID of the destination                                                                           |
| items.kind       | string  | Kind of destination. eg. kubernetes or ssh or postgres                                          |
| items.lastSeen   | string  | formatted as an RFC3339 date-time                                                               |
| items.name       | string  | Name of the destination                                                                         |
| items.resources  | array   | Destination specific. For Kubernetes, it is the list of namespaces                              |
| items.roles      | array   | Destination specific. For Kubernetes, it is the list of cluster roles available on that cluster |
| items.uniqueID   | string  | Unique ID generated by the connector                                                            |
| items.updated    | string  | formatted as an RFC3339 date-time                                                               |
| items.version    | string  | Application version of the connector for this destination                                       |

### Get a destination

```
GET /api/destinations/{id}
```

Gets the destination with the specified IDs

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Path Parameters

| Field | Format | Description                       |
| ----- | ------ | --------------------------------- |
| id    | string | ID of the destination to retrieve |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/destinations/4yJ3n3D8E2 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns an object describing the destination

#### Example response

```json
{
  "connected": true,
  "connection": {
      url: aa60eexample.us-west-2.elb.amazonaws.com,
      ca: -----BEGIN CERTIFICATE-----
          MIIDNTCCAh2gAwIBAgIRALRetnpcTo9O3V2fAK3ix+c
          -----END CERTIFICATE-----
  },
  "created": "2022-03-14T09:48:00Z",
  "id": "7a1b26b33F",
  "kind": "kubernetes",
  "lastSeen": "2022-03-14T09:48:00Z",
  "name": "production-cluster",
  "resources": ['default', 'kube-system'],
  "roles": ['cluster-admin', 'admin', 'edit', 'view', 'exec', 'logs', 'port-forward'],
  "uniqueID": "94c2c570a20311180ec325fd56",
  "updated": "2022-03-14T09:48:00Z",
  "version": "0.18.1"
}

```

#### Example response parameters

| Field          | Type    | Description                                                                                     |
| -------------- | ------- | ----------------------------------------------------------------------------------------------- |
| connected      | boolean | Shows if the destination is currently connected                                                 |
| connection.url | string  | URL for the destination                                                                         |
| connection.ca  | string  | CA for the destination                                                                          |
| created        | string  | formatted as an RFC3339 date-time                                                               |
| id             | string  | ID of the destination                                                                           |
| kind           | string  | Kind of destination. eg. kubernetes or ssh or postgres                                          |
| lastSeen       | string  | formatted as an RFC3339 date-time                                                               |
| name           | string  | Name of the destination                                                                         |
| resources      | array   | Destination specific. For Kubernetes, it is the list of namespaces                              |
| roles          | array   | Destination specific. For Kubernetes, it is the list of cluster roles available on that cluster |
| uniqueID       | string  | Unique ID generated by the connector                                                            |
| updated        | string  | formatted as an RFC3339 date-time                                                               |
| version        | string  | Application version of the connector for this destination                                       |

## Access Keys

**Access Keys** are used by automated processes to access Infra resources. To create and delete access keys, you must use the CLI or the dashboard.

### List all access keys

```
GET /api/access-keys
```

Gets a list of all access keys that meet the optional query

#### Header Parameters

| Field         | Format | Description                        |
| ------------- | ------ | ---------------------------------- |
| Infra-Version | string | Version of the API being requested |
| Authorization | string | Bearer followed by your access key |

#### Query Parameters

| Field       | Format  | Description                                            |
| ----------- | ------- | ------------------------------------------------------ |
| userID      | string  | UserID of the user whose access keys you want to list  |
| name        | string  | Name of the user                                       |
| showExpired | boolean | Whether to show expired access keys. Defaults to false |
| page        | integer | Page number to retrieve                                |
| limit       | integer | Number of objects to retrieve per page (up to 1000)    |

#### Example Request

```shell
curl -X GET https://api.infrahq.com/api/access-keys?userID=4yJ3n3D8E2 \
    &name=john@example.com \
    &showExpired=true \
    &page=1 \
    &limit=100 \
  -H 'Content-Type: application/json' \
  -H 'Infra-Version: 0.18.1' \
  -H 'Authorization: Bearer ACCESSKEY'
```

#### Returns

Returns an array of access key objects.

#### Example response

```json
{
  "count": "100",
  "items": [
    {
      "created": "2022-03-14T09:48:00Z",
      "expires": "2022-03-14T09:48:00Z",
      "extensionDeadline": "2022-03-14T09:48:00Z",
      "id": "4yJ3n3D8E2",
      "issuedFor": "4yJ3n3D8E2",
      "issuedForName": "admin@example.com",
      "lastUsed": "2022-03-14T09:48:00Z",
      "name": "cicdkey",
      "providerID": "4yJ3n3D8E2"
    }
  ],
  "limit": 100,
  "page": 1,
  "totalCount": 485,
  "totalPages": 5
}
```

#### Example response parameters

| Field                   | Type   | Description                                                   |
| ----------------------- | ------ | ------------------------------------------------------------- |
| items.created           | string | formatted as an RFC3339 date-time                             |
| items.expires           | string | key is no longer valid after this time                        |
| items.extensionDeadline | string | key must be used within this duration to remain valid         |
| items.id                | string | ID of the access key                                          |
| items.issuedFor         | string | ID of the user the key was issued to                          |
| items.issuedForName     | string | Name of the user the key was issued to                        |
| items.lastUsed          | string | formatted as an RFC3339 date-time                             |
| items.name              | string | Name of the access key                                        |
| items.providerID        | string | ID of the provider if the user is managed by an OIDC provider |
