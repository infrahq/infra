---
title: Security
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

## Encryption At Rest

Sensitive data is always encrypted at rest.
