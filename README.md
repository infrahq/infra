<p align="center">
  <img alt="logo" src="https://user-images.githubusercontent.com/3325447/162053538-b497fc85-11d8-4fb2-b43e-11db2fd0829a.png" />
</p>

# Infra

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

## Get Started

* [Quickstart](https://infrahq.com/docs/getting-started/quickstart)

## Learn More 
* [Why Infra](https://infrahq.com/docs/getting-started/introduction)
* [Key concepts](https://infrahq.com/docs/getting-started/key-concepts)
* [Security](https://infrahq.com/docs/reference/security)
