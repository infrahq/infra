# Architecture

## Infra Engine

Infra Engine is a proxy that provides access to infrastructure to users & services. It is loosely inspired by the Google [Beyondcorp Access Proxy (AP)](https://research.google/pubs/pub45728/).

### Goals
* Proxy requests to back-end infrastructure
* Easy to use CLI & Rest API
* Single binary for Linux, macOS, Windows, with easy deployment via Docker, Kubernetes
* Distribute credentials to users & services based on identity (OIDC, Service Name, etc)
* Authenticate users & services based on credentials
* Authorize users & services based roles & permissions
* Log every request for audit purposes
* [Later] Support for multiple protocols (e.g. PostgreSQL, SSH, MongoDB, etc)

### Deployment methods
* Kubernetes
* [Later] Raw binary on server
* [Later] Docker / Docker compose
* [Later] Static library for serverless / mobile / client use cases
* [Later] Cloud-hosted

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
* Single directory to federate identity, roles, permissions across Infra Engines
* Secret storage for multiple Infra Engines
* Distribute credentials across multiple Infra Engines

### Open questions:
* Should Infra Registry be a "mode" for Infra Engine vs a separate product? (think: Docker Engine in "Swarm Mode")

