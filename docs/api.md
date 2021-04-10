# API Documentation

## Contents

- [Authentication](#authentication)
- [Users](#users)
- [Tokens](#tokens)

## Authentication

Infra Engine uses API keys to authenticate requests.

### Finding your API key

```
kubectl get secret/infra-sk --template={{.data.sk}} --namespace infra | base64 -d
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

* `username` (required)

**Example**

```
curl https://api.infrahq.com/v1/users \
  -d username="testuser"
```

**Response**

```
{
  id: "us_910dj1208jd1082jd810",
  object: "user",
  username: "testuser"
}
```

### Retrieve a user

* **URL:** `/v1/users/:id`
* **Method:** GET
* **Auth Required:** Yes

**Example**

```
curl https://api.infrahq.com/v1/users/us_910dj1208jd1082jd810 \
  -u "sk_alsngunbznbmcn91u9uesdcionsdlkn38"
```

Response

```
{
  [
    { username: "testuser" }
  ]
}
```


### Delete a user

* **URL:** `/v1/users/:id`
* **Method:** DELETE
* **Auth Required:** Yes

### List users

* **URL:** `/v1/users`
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

## Tokens

Tokens are used to provide **user** access. Token format is a standard signed JWT (JSON Web Tokens) with the following claims:

```
{
  user: "us_29kf02j3a0i291k",
  exp: 1516239022 # expiry date
}
```

### Endpoints

```
  POST /v1/tokens
```

### Create a token
* **URL:** `/v1/tokens`
* **Method:** POST
* **Auth Required:** No

**Parameters**

* `password` if logging in via password

**Example 1: Password login**

```
curl https://api.inrahq.com/v1/tokens \
  -d username="testuser"
  -d password="testpassword"
```

Response:
```
{
  token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoidXNfMWlqMmQxajJpZGoxMjkiLCJleHAiOjE1MTYyMzkwMjJ9.qmUwklTyKkE6uFpVylNdQc6NLpjcqxsiH7uYPBA_c6E"
}
```

**Example 2: SSO login**

```
curl https://api.inrahq.com/v1/tokens \
  -d username="testuser"
```

Response:
```
{
  sso_url: "https://accounts.google.com/o/oauth2/v2/auth?scope=https%3A//www.googleapis.com/auth/drive.metadata.readonly&access_type=offline&include_granted_scopes=true&response_type=code&state=state_parameter_passthrough_value&redirect_uri=https%3A//oauth2.example.com/code&client_id=client_id"
}
```

**Example 3: Refresh a token**

```
curl https://api.inrahq.com/v1/tokens \
  -H "Authentication: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoidXNfMWlqMmQxajJpZGoxMjkiLCJleHAiOjE1MTYyMzkwMjJ9.qmUwklTyKkE6uFpVylNdQc6NLpjcqxsiH7uYPBA_c6E"
```

Response:
```
{
  token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoidXNfMWlqMmQxajJpZGoxMjkiLCJleHAiOjE1MTYyNDAxOTJ9.oNdZ_Yh5tdCuovzggdjbuqf6CWttiOoMzbiojU0B76Q"
}
```
