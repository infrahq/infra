# Architecture

## Infra Engine

Infra Engine is a proxy that provides access to infrastructure to users & services. It is loosely inspired by the Google [Beyondcorp Access Proxy (AP)](https://research.google/pubs/pub45728), the [Cloud SQL Auth proxy](https://cloud.google.com/sql/docs/postgres/sql-proxy) and other similar projects.

### Goals
* Distribute credentials to users & services based on identity (OIDC, Service Name, etc)
* Proxy requests from users & services to back end infrastructure
* Authenticate & authorize users & services based on credentials
* Log every request for audit purposes
* Easy to use CLI & Rest API
* Single binary for Linux, macOS, Windows, with easy deployment via Docker, Kubernetes
* [Later] Support for multiple protocols (e.g. SSH, PostgreSQL, MongoDB, etc)
* [Later] Support for encrypted point-to-point tunnels for accessing infrastructure behind private networks or NATs.

### Deployment methods
* Kubernetes
* [Later] Raw binary on server
* [Later] Docker / Docker compose
* [Later] Cloud-hosted
* [Later] Static library for serverless / mobile / client use cases

### Technology used
* Go
* [Envoy](https://www.envoyproxy.io/) 
* SQLite (Later: PostgreSQL)

How Infra works with Envoy:
* File configuration
* Shell out to Envoy (Later: include it as a binary)
* All data flows through Envoy for performance

![engine](https://user-images.githubusercontent.com/251292/113945009-7dd5f680-97d3-11eb-8835-4debaa5a1f9f.png)

### Example 1: User accessing Kubernetes

1. User logs in via CLI `infra login <Infra Engine Hostname>` (hitting the login endpoint i.e. `/v1/login`)
2. Infra engine verifies password sends user via OIDC flow (including MFA + SSO) and Infra Engine provides a JWT token upon successful login
3. Infra CLI generates a local KubeConfig entry for the cluster
4. User makes a request to access the cluster, i.e. `kubectl get pods` which hits `https://<master node endpoint>/v1/namespaces/infra/services/infra-service/proxy/v1/namespaces/default/pods` via the Infra Engine with `-H "Authorization: Bearer {JWT token}"`
5. Infra Engine verifies and extracts the user and group from the JWT token
6. Infra Engine authorizes the user based on user or group role mappings by making the upstream request using [User Impersonation](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation)
7. Infra Engine proxies the upstream request on behalf of the user and returns the result
8. The entire request lifecycle is logged to stdout


### Example 2: Container service accessing a PostgresSQL database

1. Developer creates a service identity in Infra Engine specifying the application name (or label) for which to grant access by creating a service identity `PUT /v1/services` or specifying in Infra App's YAML configuration
2. Infra Engine creates and injects a secret file (or environment variables) with generated database credentials in the container at startup via a [mutating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
3. The container application uses generated credentials to access Infra Engine via a local Kubernetes Service
4. Infra Engine verifies the credentials provided by the container
5. Infra Engine authorizes the service by role by making the upstream request to the database using [Session Authorization](https://www.postgresql.org/docs/10/sql-set-session-authorization.html)
6. Infra Engine proxies the upstream request on behalf of the user and returns the result
7. The entire request lifecycle is logged to stdout

### Open Questions
* How do we proxy access to multiple back-ends? Hostname headers? How would that work for other protocols such as SSH, MySQL, PostgreSQL?
* What if the cluster is compromised? How do we avoid storing root secrets in the Infra Engine? How could Infra Registry help with this?


## Infra Registry
Infra Registry is a centralized service for collaboration and managing 2+ Infra Engines

### Goals:
* Single directory to federate identity, roles, permissions across Infra Engines
* Secret storage for multiple Infra Engines
* No separate product from Infra Engine - instead a "registry mode" (think Docker Engine and "swarm mode")

