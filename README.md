<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/251292/179098559-75b53555-e389-40cc-b910-0e53521efad2.svg">
    <img alt="logo" src="https://user-images.githubusercontent.com/251292/179098561-eaa231c1-5757-40d7-9e5f-628e5d9c3e47.svg">
  </picture>
</div>

<div align="center">

[![GitHub release (latest by date)](https://img.shields.io/github/v/release/infrahq/infra?color=brightgreen)](https://github.com/infrahq/infra/releases/latest) [![GitHub closed issues](https://img.shields.io/github/issues-closed/infrahq/infra?color=green)](https://github.com/infrahq/infra/issues) [![GitHub commit activity](https://img.shields.io/github/commit-activity/m/infrahq/infra)](https://github.com/infrahq/infra/commits/main)
<br />
[![YouTube Channel Views](https://img.shields.io/youtube/channel/views/UCft1MzQs2BJdW8BIUu6WJkw?style=social)](https://www.youtube.com/channel/UCft1MzQs2BJdW8BIUu6WJkw) [![GitHub Repo stars](https://img.shields.io/github/stars/infrahq/infra?style=social)](https://github.com/infrahq/infra/stargazers) [![Twitter Follow](https://img.shields.io/twitter/follow/infrahq?style=social)](https://twitter.com/infrahq)

</div>

## Introduction

Infra manages access to infrastructure such as Kubernetes, with support for [more connectors](#connectors) coming soon.

- **Discover & access** infrastructure via a single command: `infra login`
- **No more out-of-sync credentials** for users (e.g. Kubeconfig)
- **Okta, Google, Azure AD** identity provider support for onboarding and offboarding
- **Fine-grained** access to specific resources that works with existing RBAC rules
- **API-first design** for managing access as code or via existing tooling
- **Temporary access** to coordinate access with systems like PagerDuty (coming soon)
- **Audit logs** for who did what, when to stay compliant (coming soon)

![dashboard](https://user-images.githubusercontent.com/251292/179115227-d7bd9040-75bc-421d-87bf-4462a4fca38d.png)

## Install

Install Infra via `helm`:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm install infra infrahq/infra
```

Next, find the Infra sign-up endpoint:

```
INFRA_HOST=$(kubectl get services/infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}") && echo "https://"$INFRA_HOST"/signup"
```

Open this URL in your web browser to get started

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

- [Login via Infra CLI](https://infrahq.com/docs/configuration/logging-in)
- [Helm Chart Reference](https://infrahq.com/docs/reference/helm-reference)
- [What is Infra?](https://infrahq.com/docs/getting-started/what-is-infra)
- [Architecture](https://infrahq.com/docs/reference/architecture)
- [Security](https://infrahq.com/docs/reference/security)

## Community

- [Community Forum](https://github.com/infrahq/infra/discussions) Best for: help with building, discussion about infrastructure access best practices.
- [GitHub Issues](https://github.com/infrahq/infra/issues) Best for: bugs and errors you encounter using Infra.
