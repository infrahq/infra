# Introduction

<div align="center">
    ![introduction](../images/InfraGithub.png)
</div>

Infra is an open-source identity management & authentication service that manages secure access to Kubernetes, Databases and internal admin tooling.

Today, teams are between a rock and a hard place when it comes to managing access to their infrastructure.

## Key Ideas

### 1. Distribute short-lived, scoped credentials to users

Infra issues short-lived credentials **unique** to each identity and only give them access to the data they need.

### 2. Work with existing identity providers

### 3. Integrate deeply with infrastructure

### 4. Extensible

Infra includes a REST API so you can extend it. For example:

* Just-in-time access
* Custom CLI tools to access infrastructure

## How it works

![architecture](../images/architecture.svg)

### Infra Server & API

At the core of Infra is an API for handling

### Connectors

Connectors are lightweight services that run alongside infrastructure such as Kubernetes.

### Infra CLI

## Concepts

### Identities

Identities are users that represent a human or machines in the real world. Identities can be organized into **Groups**.

### Destinations

### Providers

### Grants

### Keys
