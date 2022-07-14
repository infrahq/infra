<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/251292/179056556-361358af-aab9-4096-a714-87184f1afb22.svg">
    <img alt="logo" src="https://user-images.githubusercontent.com/251292/179056708-48e3c20b-22d1-4a40-9860-2f120c52a34f.svg">
  </picture>
</p>

![GitHub commit checks state](https://img.shields.io/github/checks-status/infrahq/infra/main?label=Build) ![GitHub closed issues](https://img.shields.io/github/issues-closed/infrahq/infra?color=green) ![GitHub commit activity](https://img.shields.io/github/commit-activity/m/infrahq/infra) ![GitHub Repo stars](https://img.shields.io/github/stars/infrahq/infra?style=social) ![Twitter Follow](https://img.shields.io/twitter/follow/infrahq?style=social)

## Introduction

Infra is **open-source access management** for infrastructure (Kubernetes, SSH, Databases, AWS and more):

1. Connect infrastructure (e.g. Kubernetes, SSH, Databases, AWS and more coming soon)
2. Add an identity provider such as Google, Okta or Azure AD, or use Infra's built-in users & groups
3. Assign access (e.g. `view`, `edit` or `admin`) to users & groups on the team
4. Users log in via Infra's CLI with `infra login`

That's it!

![dashboard](https://user-images.githubusercontent.com/251292/179054958-cba0e177-dd35-42ea-ad28-a6c8a79e697a.png)

### Features

- **Discover & access** all infrastructure in one place
- **No more out of sync credentials** or user configurations: Infra automatically rotates credentials
- **Support for native RBAC**: edit, view and even support custom roles like `exec`
- **Onboard and offboard users via an identity provider** (Okta, Azure AD, Google or OIDC)
- **Management dashboard** for easy management
- **Audit logs**: see who did what, when to stay compliant (coming soon)

## Get Started

Deploy Infra

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

- [Log in via Infra CLI](https://infrahq.com/docs/configuration/logging-in)
- [What is Infra?](https://infrahq.com/docs/getting-started/what-is-infra)
- [Architecture](https://infrahq.com/docs/reference/architecture)
- [Security](https://infrahq.com/docs/reference/security)
- [Helm Chart Reference](https://infrahq.com/docs/reference/helm-reference)

## Community

- [Community Forum](https://github.com/infrahq/infra/discussions) Best for: help with building, discussion about infrastructure access best practices.
- [GitHub Issues](https://github.com/infrahq/infra/issues) Best for: bugs and errors you encounter using Infra.
