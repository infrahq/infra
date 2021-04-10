# API Documentation

## Contents

- [Authentication](#authentication)
- [Users](#users)
- [Tokens](#tokens)

## Authentication

Infra Engine uses API keys to authenticate requests.

### Finding your API key

```
kubectl get secret/infra-sk  --template={{.data.sk}} --namespace infra | base64 -d
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
* **Method:** PUT
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

Tokens are used to provide **users** access to infrastructure.

### Endpoints

```
  POST /v1/tokens
```

### Create a token
* **URL:** `/v1/tokens/`
* **Method:** POST
* **Auth Required:** No

**Parameters**

* `password` if logging in via password

**Example**

```
curl https://api.inrahq.com/v1/tokens \
  -d username="testuser"
  -d password="testpassword"
```

Response:
```
{
  token: "ja781pubnsqckjboa6gdaoiy2dbap2dap27dha[28dhapsyfgh97qph2dh12d71hgg98723dnks;ljdjal;sdkjf;3hj08fu"
}
```

Response (if using SSO):
```
{
  sso_url: "https://accounts.google.com/o/oauth2/v2/auth?scope=https%3A//www.googleapis.com/auth/drive.metadata.readonly&access_type=offline&include_granted_scopes=true&response_type=code&state=state_parameter_passthrough_value&redirect_uri=https%3A//oauth2.example.com/code&client_id=client_id"
}
```
