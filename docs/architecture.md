---
id: architecture
title: Architecture
sidebar_label: Architecture
slug: /architecture
---

# Architecture

## Infra Engine

Infra Engine is designed as an identity-aware proxy that runs adjacent to the infrastructure being accessed.

Key challenges:
* Proxy requests to back-end infrastructure
* Authenticate users & services based on identity 
* Securely store root credentials for back-end infrastructure being accessed
* Log every request and collect metrics
* [Later] Support of multiple protocols (HTTP, Postgres, MongoDB, etc)
* [Later] Tunneling support for accessing infrastructure behind VPNs

## Infra Registry

Infra registry is designed to help teams adopt multiple Infra engines across their team.

Key challenges:
* Centralize identity data from one more providers
* Secrets storage
* Collet

