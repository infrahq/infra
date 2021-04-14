# API Reference

## Contents
- [Authenticating](#authenticating)
- [Sources](#sources)
- [Destinations](#destinations)
- [Credentials](#credentials)

## Authenticating

To authenticate with Infra, log in using 

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
* `password` (optional)
* `pod` (optional) the pod name

**Example 1: Person**

```bash
curl https://api.infrahq.com/v1/users \
  -d name="testuser" \
  -d password="mypassword" 
```

Response:

```json
{
  id: "src_910dj1208jd1082jd810",
  object: "source",
  username: "testuser"
}
```

**Example 2: Kubernetes Pod**

```bash
curl https://api.infrahq.com/v1/users \
  -d name="app" \
  -d pod="app"
```

Response:

```json
{
  id: "src_a0s8jfws08jfs038s038j",
  object: "source",
  pod: "app"
}
```

### Retrieve a source

* **URL:** `/v1/sources/:id`
* **Method:** GET
* **Auth Required:** Yes

**Example**

```bash
curl https://api.infrahq.com/v1/s/src_a0s8jfws08jfs038s038j
```

Response

```json
{
  [
    {
      id: "src_a0s8jfws08jfs038s038j",
      object: "source",
      pod: "app"
    }
  ]
}
```


### Delete a source

* **URL:** `/v1/source/:id`
* **Method:** DELETE
* **Auth Required:** Yes

**Example**

```


### List users

* **URL:** `/v1/sources`
* **Method:** GET
* **Auth Required:** Yes

Response

```
{
  [
    { username: "testuser1" },
    { username: "testuser2" }
  ]
}
```

## Credentials

Credentials grant access to a destination

### Endpoints

```
  POST /v1/creds
```

### Create a credential
* **URL:** `/v1/creds`
* **Method:** POST
* **Auth Required:** Yes

**Parameters**

* `password` if logging in via password

**Example 1: Kubernetes**

```
curl https://api.infrahq.com/v1/creds \
  -d destination="production_cluster" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoidXNfMWlqMmQxajJpZGoxMjkiLCJleHAiOjE1MTYyMzkwMjJ9.qmUwklTyKkE6uFpVylNdQc6NLpjcqxsiH7uYPBA_c6E"
```

Response:
```
{
  token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoidXNfMWlqMmQxajJpZGoxMjkiLCJleHAiOjE1MTYyMzkwMjJ9.qmUwklTyKkE6uFpVylNdQc6NLpjcqxsiH7uYPBA_c6E"
}
```

**Example 2: SSH**
