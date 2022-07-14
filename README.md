<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/251292/179072134-f520904a-ccb8-44aa-9ca0-cecfa4eabe11.svg">
    <img alt="logo" src="https://user-images.githubusercontent.com/251292/179072481-45a81045-161b-4491-8578-5f5a386a9b18.svg">
  </picture>
</p>

![GitHub commit checks state](https://img.shields.io/github/checks-status/infrahq/infra/main?label=Build) [![GitHub closed issues](https://img.shields.io/github/issues-closed/infrahq/infra?color=green)](https://github.com/infrahq/infra/issues) [![GitHub commit activity](https://img.shields.io/github/commit-activity/m/infrahq/infra)](https://github.com/infrahq/infra/commits/main) [![YouTube Channel Views](https://img.shields.io/youtube/channel/views/UCft1MzQs2BJdW8BIUu6WJkw?style=social)](https://www.youtube.com/channel/UCft1MzQs2BJdW8BIUu6WJkw) [![GitHub Repo stars](https://img.shields.io/github/stars/infrahq/infra?style=social)](https://github.com/infrahq/infra/stargazers) [![Twitter Follow](https://img.shields.io/twitter/follow/infrahq?style=social)](https://twitter.com/infrahq)

## Introduction

Manage access to Kubernetes, SSH, Databases, AWS and more:

1. Connect your infrastructure
2. Add your team via Google, Okta or Azure AD or using Infra's built-in user management
3. Assign fine-grained access (e.g. `view`, `edit` or `admin`) to any user or group

**That's it!** Your team has access via Infra's CLI `infra login`

## Get Started

Deploy Infra via `helm`:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm install infra infrahq/infra
```

## Features

![dashboard](https://user-images.githubusercontent.com/251292/179054958-cba0e177-dd35-42ea-ad28-a6c8a79e697a.png)

- **Discover & access** infrastructure via a single command: `infra login`
- **No more out-of-sync credentials** for users (e.g. Kubeconfig)
- **Okta, Google, Azure AD** identity provider support for onboarding and offboarding
- **Fine-grained** access to specific resources that works with existing RBAC rules
- **API-first design** for managing access as code or via existing tooling

Coming soon:

- **Dynamic access** to coordinate access with systems like PagerDuty
- **Audit logs** for who did what, when to stay compliant

## Connectors

| Connector          | Status        | Documentation                                          |
| ------------------ | ------------- | ------------------------------------------------------ |
| Kubernetes         | ✅ Stable     | [Link](https://infrahq.com/docs/connectors/kubernetes) |
| Postgres           | _Coming soon_ |                                                        |
| SSH                | _Coming soon_ |                                                        |
| AWS                | _Coming soon_ |                                                        |
| Container Registry | _Coming soon_ |                                                        |
| MongoDB            | _Coming soon_ |                                                        |
| Snowflake          | _Coming soon_ |                                                        |
| MySQL              | _Coming soon_ |                                                        |
| RDP                | _Coming soon_ |                                                        |

## Documentation

- [Log in via Infra CLI](https://infrahq.com/docs/configuration/logging-in)
- [What is Infra?](https://infrahq.com/docs/getting-started/what-is-infra)
- [Architecture](https://infrahq.com/docs/reference/architecture)
- [Security](https://infrahq.com/docs/reference/security)
- [Helm Chart Reference](https://infrahq.com/docs/reference/helm-reference)

## Community

- [Community Forum](https://github.com/infrahq/infra/discussions) Best for: help with building, discussion about infrastructure access best practices.
- [GitHub Issues](https://github.com/infrahq/infra/issues) Best for: bugs and errors you encounter using Infra.
