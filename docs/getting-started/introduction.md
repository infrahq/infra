# Introduction

<p align="center">
  <img alt="logo" src="https://user-images.githubusercontent.com/3325447/162053538-b497fc85-11d8-4fb2-b43e-11db2fd0829a.png" />
</p>

Infra enables you to **discover and access** infrastructure (e.g. Kubernetes, databases). We help you connect an identity provider such as Okta or Azure active directory, and map users/groups with the permissions you set to your infrastructure. 

If you don't have an identity provider, Infra supports local users for you to get started before connecting an identity provider. 

## Features

* Single-command to discover & access all your infrastructure (as an example, for Kubernetes, Infra automatically creates and syncs your kubeconfig locally after `infra login` and gets out of your way so you can use your favorite tools to access it) 
* No more out-of-sync user configurations no matter where your clusters are hosted 
* Support for native RBAC (e.g. support for default Kubernetes cluster roles or mapping to your own existing cluster roles)  
* Onboard and offboard users via an identity provider (e.g. Okta) 
* Workflow for dynamically requesting & granting access to users (coming soon) 
* Audit logs for who did what, when (coming soon) 

<p align="center">
  <img alt="product screenshot" src="https://user-images.githubusercontent.com/3325447/162065853-0073e6f2-8094-42f4-b88b-1bf03b2264e0.png"  />
</p>

## Key Ideas

### 1. Distribute short-lived, scoped credentials to users

Infra issues short-lived credentials **unique** to each identity and only give them access to the data they need.

### 2. Work seamlessly with existing identity providers


### 3. Plug & play connectors for Kubernetes & more

Infra integrates deeply with infrastructure such as Kubernetes

### 4. Extensible

Infra includes a REST API so you can extend it. For example:

* Just-in-time access
* Custom CLI tools to access infrastructure
* Coordinating access with on-call rotation

## Architecture

![architecture](../images/architecture.svg)

## Concepts
