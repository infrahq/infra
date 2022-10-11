---
title: Security
position: 5
---

# Security

We take security very seriously. This document provides an overview of Infra's security model. If you have found a security vulnerability, please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com).

## General Security

### HTTPS

By default, Infra and its components communicate via TLS. Infra's API server generates self-signed TLS certificates out of the box, and valid public or private TLS certificates can be used with the Infra server by putting it behind a Kubernetes ingress.

### Authentication

When users login to Infra as a valid user, they are issued a session token with a system generated 24 character secret. Infra stores he SHA256 hash of this token for token validation. This session token is available locally under `~/.infra`.

When a user connects to a cluster after login, Infra issues a new JWT signed with an ECDSA signature using P-521 and SHA-512. The connector verifies this JWT. If the JWT and the user role is valid at the destination, the user is granted access.

## Deployment

When deploying Infra, we recommend Infra be deployed in its own namespace to minimize the deployment scope.

## Sensitive Information

Secrets can be stored in a variety of [secret storage backends](./helm.md#secrets), including Kubernetes secrets, Vault, AWS Secrets Manager, AWS SSM (Systems Manager Parameter Store), and some simple options exist for loading secrets from the OS or container, such as: loading secrets from environment variables, loading secrets from files on the file system, and even plaintext secrets directly in the configuration file (though this is not recommended). With all types except for `plaintext`, the configuration file lists the respective secret object names and Infra's storage never persists the actual secret. In the case of all secret types (including `plaintext`), the database [encrypts at rest](#encrypted-at-rest) any secret data.

## Encrypted At Rest

Sensitive data is always encrypted at rest. See [encryption keys](./helm.md#encryption-keys) for more information on how to use custom encryption keys with Infra.
