<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/251292/179030679-e298d1c5-0933-4338-988f-c9785442335b.svg">
    <img alt="logo" src="https://user-images.githubusercontent.com/251292/179030550-27b8cdda-07ec-48e6-ba41-04f21425738b.svg">
  </picture>
</p>

![GitHub commit checks state](https://img.shields.io/github/checks-status/infrahq/infra/main?label=Build) ![GitHub closed issues](https://img.shields.io/github/issues-closed/infrahq/infra?color=green) ![GitHub commit activity](https://img.shields.io/github/commit-activity/m/infrahq/infra) ![GitHub Repo stars](https://img.shields.io/github/stars/infrahq/infra?style=social) ![Twitter Follow](https://img.shields.io/twitter/follow/infrahq?style=social)

## Introduction

Infra is open-source access management for infrastructure (Kubernetes, SSH, Databases, AWS and more).

## Features

![dashboard](https://user-images.githubusercontent.com/251292/179054958-cba0e177-dd35-42ea-ad28-a6c8a79e697a.png)

- **Discover & access** infrastructure in one place
- **No more out of sync credentials** or user configurations: Infra automatically rotates credentials
- **Support for native RBAC**: edit, view and even support custom roles like `exec`
- **Onboard and offboard users via an identity provider** (Okta, Azure AD, Google or OIDC)
- **Audit logs**: see who did what, when to stay compliant (coming soon)

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
