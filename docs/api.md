# API Reference

## Contents

### Overview
- [Authentication](#authentication)
- [Pagination](#authentication)
- [Referencing Secrets](#secrets)

### Core Resources
- [Users](#users)
- [Groups](#groups)
- [Roles](#roles)
- [Permissions](#permissions)

## Authentication

### Endpoints

```
POST /v1/login
```

### Login

* **URL:** `/v1/login`
* **Method:** POST
* **Auth Required:** Yes

* `username` (required)

**Example**

```bash
curl https://api.infrahq.com/v1/login \
  -d username="testuser"
```

Response:

```json
{
  "sso_url": "https://example.okta.com/login..."
}
```

## Users

### Endpoints

```
  POST /v1/users
   GET /v1/users/:id
DELETE /v1/users/:id
   GET /v1/users
```

### Create a user

* **URL:** `/v1/users`
* **Method:** POST
* **Auth Required:** Yes

**Parameters**

* `username` (optional)
* `password`

**Example**

```bash
curl https://api.infrahq.com/v1/users \
  -d username="testuser" \
  -d password="mypassword"
```

Response:

```json
{
  "id": "usr_910dj1208jd1082jd810",
  "username": "testuser"
}
```

### Retrieve a user

* **URL:** `/v1/users/:id`
* **Method:** GET
* **Auth Required:** Yes

**Example**

```bash
curl https://api.infrahq.com/v1/users/usr_910dj1208jd1082jd810
```

Response

```json
{
  "id": "usr_910dj1208jd1082jd810",
  "object": "user",
  "username": "testuser"
}
```

### Delete a user

* **URL:** `/v1/users/:id`
* **Method:** DELETE
* **Auth Required:** Yes

**Example**

```
curl -X DELETE https://api.infrahq.com/v1/users/usr_a0s8jfws08jfs038s038j
```

Note that if this source has been imported via an identity provider, they continue to be imported and updated, but will remain in a blacklist.

### List users

* **URL:** `/v1/users`
* **Method:** GET
* **Auth Required:** Yes

**Example**

```
curl https://api.infrahq.com/v1/users
```

Response

```json
{
  "object": "list",
  "url": "/v1/users",
  "has_more": false,
  "data": [
    {
      "object": "users",
      "id": "usr_910dj1208jd1082jd810",
      "name": "testuser"
    }
  ]
}
```
