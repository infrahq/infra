# Security
We take security very seriously. This document provides an overview of Infra's security model.

If you have found a security vulnerability please disclose it privately to us by email via security@infrahq.com.

## General Security
### HTTPS
By default Infra and Infra engine communicate via encrypted HTTPS connections with validated certificates. When using self-signed certificates, an error will be printed in the logs. Certificate validation can be strongly enforced using the `--force-tls-verify` flag.

### Authentication
When users login to Infra as a valid user they are issued a session token with a 24 character secret that is randomly generated. The SHA256 hash of this token is stored server-side for token validation. This session token is stored locally under `~/.infra`.

When a user connects to a cluster after login, Infra issues a new JWT signed with an ECDSA signature using P-521 and SHA-512. This JWT is verified by the engine. If JWT and the user role is valid at the destination, the user is granted access.

## Deployment
When deploying Infra, we recommend Infra be deployed in its own namespace to minimize the deployment scope. 

## Sensitive Information

### Okta secrets
Infra uses an Okta application client secret and API token in order to allow users to authenticate via an OpenID Connect (OIDC) authorization code flow. These secrets are stored using Kubernetes secrets. Their respective secret object names are specified in the configuration file and the actual secret is never persisted in Infra's storage. 

#### Okta client secret usage:
The client secret is loaded server-side from the specified Kubernetes secret only when a user is logging in via Okta.

#### Okta API token usage:
The Okta API token is only used for read actions. It is retrieved from the kubernetes secret when validating that the Okta connection is valid and when syncing users/groups.
