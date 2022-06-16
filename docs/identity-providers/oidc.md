---
title: OIDC
position: 4
---

# OIDC

## Connecting an OpenID Connect (OIDC) Identity Provider

To connect an OIDC identity provider, run the following command:

```
infra providers add <your oidc provider name> \
  --url <your oidc provider url (or domain)> \
  --client-id <your oidc client id> \
  --client-secret <your oidc client secret>
```


## Finding required values

### OIDC Provider Name
This can be any value you desire. It is used as a name in Infra to refer to this identity provider.

### OIDC Provider URL
The base URL your OIDC identity provider can be reached at to obtain information and perform authentication.

Infra relies on the [/.well-known/openid-configuration](https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderConfig) endpoint to discover the paths needed to use the identity provider.

For example, if your OIDC provider's discovery endpoint is `https://oidc.example.com/.well-known/openid-configuration` then your OIDC provider URL would be `oidc.example.com`.

### OIDC Client ID and Secret
In order to authenticate using an OIDC identity provider you must register Infra as a client in that identity provider. By registering Infra as a client it will be granted a client ID and client secret that it can use to authenticate users.

**OIDC Client Configuration Requirements**
- Infra uses the [authorization code flow](https://openid.net/specs/openid-connect-core-1_0.html#CodeFlowAuth), typically clients that use this flow are considered **web applications**.
- Scopes required:
  - `openid`
  - `email`
- Redirect URIs:
  - `http://localhost:8301` (for Infra CLI)
  - `https://<INFRA_SERVER_HOST>/login/callback` (for Infra Dashboard)

## Additional Requirements
- The OIDC identity provider must support the [UserInfo](https://openid.net/specs/openid-connect-core-1_0.html#UserInfo) endpoint.
- The UserInfo response **must** contain either a `name` or `email` field.
- If you wish to use groups the identity provider **must** return the user's assigned groups from the UserInfo endpoint.
