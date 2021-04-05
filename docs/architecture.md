# Architecture

## Infra Engine

Infra Engine is a proxy service for managing access to infrastructure.

### Goals
* Easy to use CLI & Rest API
* Single binary for Linux, macOS, Windows, with easy deployment on Kubernetes
* Distribute credentials to users & services based on identity (i.e. JWT tokens, etc.)
* Authenticate requests based on credentials
* Authorize requests based on identity, roles & permissions
* Proxy requests to back-end services
* Log every request for audit purposes
* [Later] Collect metrics & statistics based on identity
* [Later] Infra Registry: synchronize identities and 
* [Later] Infra Registry: Register infrastructure for discovery
* [Later] Infra Registry: Report access logs for analysis in a single places

### Deployment methods
* Kubernetes
* [Later] Raw binary
* [Later] Docker / Docker compose
* [Later] Cloud-hosted
* [Later] Static library for serverless / mobile / client use cases

### Technology used:
* Go
* Envoy, the battle-hardened [Envoy](https://www.envoyproxy.io/) proxy project was chosen as a starting point

How Infra works with Envoy:
* File configuration
* Shell out to Envoy
* [Later] Include Envoy a library

## Infra Registry

Infra Registry is a centralized service for collaboration and managing 2+ Infra Engines

### Goals:
* Single directory to federate identity across Infra Engines
* Secret storage for multiple Infra Engines
* Distribute credentials across multiple Infra Engines

### Open questions:
* Should Infra Registry be a "mode" for Infra Engine vs a separate product? (think: Docker Engine in "Swarm Mode")

