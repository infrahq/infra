# Security
We take security very seriously. This document provides an overview of Infra's security model.

If you have found a security vulnerability please disclose it privately to us by email via security@infrahq.com.

## General Security
### HTTPS
By default the Infra registry and Infra engine communicate via encrypted HTTPS connections with validated certificates. Failure to validate a certificate will by default result in a connection being aborted.

### Authentication
When users login to Infra as a valid user they are issued a session token with a 24 character secret that is randomly generated. The SHA256 hash of this token is stored server-side for token validation when it is presented. This session token is stored in the Infra CLI.

When a user connects to a cluster after logging in their request is proxied to the Infra registry which issues them a new JWT signed with an RS256 signature. This JWT is presented to the engine and if it is valid the user is granted access if they have a valid role at the destination.

## Deploying in Production
When deploying in your production cluster ensure the Infra Registry is deployed in its own namespace. Deploying the registry in its own namespace allows you to securely manage which resources the deployment has access to.

## Sensitive Information

### Infra Registry API token
In order for an engine to establish a connection with Infra registry it must present an API token. This token is stored in a Kubernetes secret and is never persisted in Infra's storage. It may be changed at any time and applied by restarting the registry. 

### Okta secrets
Infra uses an Okta application client secret and API token in order to allow users to authenticate via an OpenID Connect (OIDC) authorization code flow. These secrets are stored using Kubernetes secrets. Their respective secret object names are specified in the configuration file and the actual secret is never persisted in Infra's storage. 

#### Okta client secret usage:
The client secret is loaded server-side from the specified Kubernetes secret only when a user is logging in via Okta.

#### Okta API token usage:
The Okta API token is only used for read actions. It is retrieved from the kubernetes secret when validating that the Okta connection is valid and when syncing users/groups.
