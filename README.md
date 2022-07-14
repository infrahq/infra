<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/251292/179030679-e298d1c5-0933-4338-988f-c9785442335b.svg">
    <img alt="logo" src="https://user-images.githubusercontent.com/251292/179030550-27b8cdda-07ec-48e6-ba41-04f21425738b.svg">
  </picture>
</p>

## Introduction

Infra enables you to **discover and access** infrastructure (e.g. Kubernetes, databases). We help you connect an identity provider such as Okta or Azure active directory, and map users/groups with the permissions you set to your infrastructure.

If you don't have an identity provider, Infra supports local users for you to get started before connecting an identity provider.

- Single-command to discover & access all your infrastructure (as an example, for Kubernetes, Infra automatically creates and syncs your kubeconfig locally after `infra login` and gets out of your way so you can use your favorite tools to access it)
- No more out-of-sync user configurations no matter where your clusters are hosted
- Support for native RBAC (e.g. support for default Kubernetes cluster roles or mapping to your own existing cluster roles)
- Onboard and offboard users via an identity provider (e.g. Okta)
- Workflow for dynamically requesting & granting access to users (coming soon)
- Audit logs for who did what, when (coming soon)

![dashboard](https://user-images.githubusercontent.com/251292/179031390-23f08bc4-96f5-4ccc-915a-da5c7c2c1256.png)

## Get Started

Deploy Infra:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm install infra infrahq/infra
```

Next, retrieve the hostname of the Infra server:

```
INFRA_SERVER=$(kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[\*]['ip', 'hostname']}" -w)
```

Next, navigate to [https://<INFRA_SERVER>](https://<INFRA_SERVER>) to open the Infra Dashboard

## Documentation

- [Quickstart](https://infrahq.com/docs/getting-started/quickstart)
- [What is Infra?](https://infrahq.com/docs/getting-started/what-is-infra)
- [Architecture](https://infrahq.com/docs/reference/architecture)
- [Security](https://infrahq.com/docs/reference/security)

## Community

- [Community Forum](https://github.com/infrahq/infra/discussions) Best for: help with building, discussion about infrastructure access best practices.
- [GitHub Issues](https://github.com/infrahq/infra/issues) Best for: bugs and errors you encounter using Infra.
