# API Reference

## Contents

### Overview
- [Authentication](#authentication)
- [Pagination](#authentication)
- [Referencing Secrets](#secrets)

### Core Resources
- [Sources](#sources)
- [Destinations](#destinations)
- [Permissions](#permissions)
- [Credentials](#credentials)


## Authentication (as a user)

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

## Sources

Sources are users or services that access infrastructure [Destinations](#destinations) via Infra.

### Endpoints

```
  POST /v1/sources
   GET /v1/sources/:id
DELETE /v1/sources/:id
   GET /v1/sources
```

### Create a source

* **URL:** `/v1/sources`
* **Method:** POST
* **Auth Required:** Yes

**Parameters**

* `name` (required)
* `username` (optional)
* `password` (optional)
* `kubernetes` (optional)
  * `kubernetes.pod` (optional) the pod name
  * `kubernetes.label` (optional) the pod label

**Example 1: Person**

```bash
curl https://api.infrahq.com/v1/sources \
  -d name="testuser" \
  -d username="testuser" \
  -d password="mypassword"
```

Response:

```json
{
  "id": "src_910dj1208jd1082jd810",
  "object": "source",
  "name": "testuser",
  "username": "testuser"
}
```

**Example 2: Kubernetes Pod**

```bash
curl https://api.infrahq.com/v1/sources \
  -d name="app" \
  -d "kubernetes.pod"="app"
```

Response:

```json
{
  "id": "src_a0s8jfws08jfs038s038j",
  "object": "source",
  "name": "app",
  "pod": "app"
}
```

### Retrieve a source

* **URL:** `/v1/sources/:id`
* **Method:** GET
* **Auth Required:** Yes

**Example**

```bash
curl https://api.infrahq.com/v1/sources/src_910dj1208jd1082jd810
```

Response

```json
{
  "id": "src_910dj1208jd1082jd810",
  "object": "source",
  "name": "testuser"
}
```


### Delete a source

* **URL:** `/v1/source/:id`
* **Method:** DELETE
* **Auth Required:** Yes

**Example**

```
curl -X DELETE https://api.infrahq.com/v1/sources/src_a0s8jfws08jfs038s038j
```

Note that if this source has been imported via an identity provider, they continue to be imported and updated, but will remain in a blacklist.

### List sources

* **URL:** `/v1/sources`
* **Method:** GET
* **Auth Required:** Yes

**Example**

```
curl https://api.infrahq.com/v1/sources
```

Response

```json
{
  "object": "list",
  "url": "/v1/sources",
  "has_more": false,
  [
    {
      "object": "source",
      "id": "src_910dj1208jd1082jd810",
      "name": "testuser"
    },
    {
      "object": "source",
      "id": "src_a0s8jfws08jfs038s038j",
      "name": "app",
      "pod": "app"
    }
  ]
}
```

## Destinations

### Endpoints

```
  POST /v1/destinations
   GET /v1/destinations/:id
DELETE /v1/destinations/:id
   GET /v1/destinations
```

### Connect a destination

* **URL:** `/v1/destinations`
* **Method:** POST
* **Auth Required:** Yes

**Parameters**

* `name` (required)
* `kubernetes` (optional) For adding Kubernetes clusters
  * `kubernetes.master` (optional) Kubernetes master node endpoint
  * `kubernetes.ca_client` (optional) Kubernetes client CA
  * `kubernetes.service_account_token` (optional) Kubernetes service account token

**Example**

```bash
curl https://api.infrahq.com/v1/sources \
  -d name="staging" \
  -d "kubernetes.master"="31.29.291.281" \
  -d "kubernetes.ca_client"=-----BEGIN CERTIFICATE-----MIIDmzCCAoOgAwIBAgIEU9e2rzANBgkqhkiG9w0B...-----END CERTIFICATE----- \
  -d "kubernetes.service_account_token"="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoidXNfMWlqMmQxajJpZGoxMjkiLCJleHAiOjE1MTYyMzkwMjJ9.qmUwklTyKkE6uFpVylNdQc6NLpjcqxsiH7uYPBA_c6E"
```

Response:

```json
{
  "id": "dst_aj08jda20dj9a2lh",
  "object": "destination",
  "username": "staging",
  "kubernetes": {
    "master": "31.29.291.281",
    "ca_client": "=-----BEGIN CERTIFICATE-----MIIDmzCCAoOgAwIBAgIEU9e2rzANBgkqhkiG9w0B...-----END CERTIFICATE-----"
}
```

### Retrieve a destination

* **URL:** `/v1/destinations/:id`
* **Method:** GET
* **Auth Required:** Yes

**Example**

```bash
curl https://api.infrahq.com/v1/destinations/dst_aj08jda20dj9a2lh
```

Response

```json
{
  "id": "dst_aj08jda20dj9a2lh",
  "object": "destination",
  "username": "staging",
  "kubernetes": {
    "master": "31.29.291.281",
    "ca_client": "=-----BEGIN CERTIFICATE-----MIIDmzCCAoOgAwIBAgIEU9e2rzANBgkqhkiG9w0B...-----END CERTIFICATE-----"
}
```


### Disconnect a destination

* **URL:** `/v1/destinations/:id`
* **Method:** DELETE
* **Auth Required:** Yes

**Example**

```
curl -X DELETE https://api.infrahq.com/v1/destinations/dst_aj08jda20dj9a2lh
```

### List destinations

* **URL:** `/v1/destinations`
* **Method:** GET
* **Auth Required:** Yes

**Example**

```
curl https://api.infrahq.com/v1/sources
```

Response

```json
{
  "object": "list",
  "url": "/v1/destinations",
  "has_more": false,
  [
    {
      "id": "dst_aj08jda20dj9a2lh",
      "object": "destination",
      "username": "staging",
      "kubernetes": {
        "master": "31.29.291.281",
        "ca_client": "=-----BEGIN CERTIFICATE-----MIIDmzCCAoOgAwIBAgIEU9e2rzANBgkqhkiG9w0B...-----END CERTIFICATE-----"
    }
  ]
}
```

## Credentials

Credentials grant access to destination to a requesting source.

### Endpoints

```
  POST /v1/creds
```

### Create a credential
* **URL:** `/v1/creds`
* **Method:** POST
* **Auth Required:** Yes

**Parameters**

* `destination` 

**Example**

```
curl https://api.infrahq.com/v1/creds \
  -d destination="dst_aj08jda20dj9a2lh" \
  -u "testuser:password"
```

Response:
```
{
  token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoidXNfMWlqMmQxajJpZGoxMjkiLCJleHAiOjE1MTYyMzkwMjJ9.qmUwklTyKkE6uFpVylNdQc6NLpjcqxsiH7uYPBA_c6E"
}
```
