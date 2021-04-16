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
   GET /v1/users
```

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

## Groups

## Roles

