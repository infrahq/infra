<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/251292/179096064-322cb0df-57dc-4d37-af49-d213dc0ad481.svg">
  <img alt="logo" src="https://user-images.githubusercontent.com/251292/179096065-7f7a9a1d-b072-4ee8-bf8a-c92c07a28e16.svg">
</picture>

![GitHub commit checks state](https://img.shields.io/github/checks-status/infrahq/infra/main?label=Build) [![GitHub closed issues](https://img.shields.io/github/issues-closed/infrahq/infra?color=green)](https://github.com/infrahq/infra/issues) [![GitHub commit activity](https://img.shields.io/github/commit-activity/m/infrahq/infra)](https://github.com/infrahq/infra/commits/main) [![YouTube Channel Views](https://img.shields.io/youtube/channel/views/UCft1MzQs2BJdW8BIUu6WJkw?style=social)](https://www.youtube.com/channel/UCft1MzQs2BJdW8BIUu6WJkw) [![GitHub Repo stars](https://img.shields.io/github/stars/infrahq/infra?style=social)](https://github.com/infrahq/infra/stargazers) [![Twitter Follow](https://img.shields.io/twitter/follow/infrahq?style=social)](https://twitter.com/infrahq)

## Introduction

Infra manages access to Kubernetes, with support for [more cloud infrastrucure](#connectors) coming soon.

1. Connect your clusters, servers, databases, and other resources
2. Add your team via Google, Okta or Azure AD or using Infra's built-in user management
3. Assign fine-grained access (e.g. `view`, `edit` or `admin`) to any user or group
4. **That's it!** Your team can now discover and access via `infra login`

## Get Started

Deploy Infra via `helm`:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm install infra infrahq/infra
```

Find the exposed hostname:

```
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}" -w
```

Visit the hostname in your browser, or run `infra login` to get up and running via the CLI.

## Features

![dashboard](https://user-images.githubusercontent.com/251292/179054958-cba0e177-dd35-42ea-ad28-a6c8a79e697a.png)

- **Discover & access** infrastructure via a single command: `infra login`
- **No more out-of-sync credentials** for users (e.g. Kubeconfig)
- **Okta, Google, Azure AD** identity provider support for onboarding and offboarding
- **Fine-grained** access to specific resources that works with existing RBAC rules
- **API-first design** for managing access as code or via existing tooling
- **Dynamic access** to coordinate access with systems like PagerDuty (coming soon)
- **Audit logs** for who did what, when to stay compliant (coming soon)

## Connectors

| Connector          | Status        | Documentation                                                 |
| ------------------ | ------------- | ------------------------------------------------------------- |
| Kubernetes         | ✅ Stable     | [Get started](https://infrahq.com/docs/connectors/kubernetes) |
| Postgres           | _Coming soon_ | _Coming soon_                                                 |
| SSH                | _Coming soon_ | _Coming soon_                                                 |
| AWS                | _Coming soon_ | _Coming soon_                                                 |
| Container Registry | _Coming soon_ | _Coming soon_                                                 |
| MongoDB            | _Coming soon_ | _Coming soon_                                                 |
| Snowflake          | _Coming soon_ | _Coming soon_                                                 |
| MySQL              | _Coming soon_ | _Coming soon_                                                 |
| RDP                | _Coming soon_ | _Coming soon_                                                 |

## Documentation

- [Log in via Infra CLI](https://infrahq.com/docs/configuration/logging-in)
- [What is Infra?](https://infrahq.com/docs/getting-started/what-is-infra)
- [Architecture](https://infrahq.com/docs/reference/architecture)
- [Security](https://infrahq.com/docs/reference/security)
- [Helm Chart Reference](https://infrahq.com/docs/reference/helm-reference)

## Community

- [Community Forum](https://github.com/infrahq/infra/discussions) Best for: help with building, discussion about infrastructure access best practices.
- [GitHub Issues](https://github.com/infrahq/infra/issues) Best for: bugs and errors you encounter using Infra.
