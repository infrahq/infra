<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/251292/179072134-f520904a-ccb8-44aa-9ca0-cecfa4eabe11.svg">
    <img alt="logo" src="https://user-images.githubusercontent.com/251292/179072481-45a81045-161b-4491-8578-5f5a386a9b18.svg">
  </picture>
</p>

![GitHub commit checks state](https://img.shields.io/github/checks-status/infrahq/infra/main?label=Build) [![GitHub closed issues](https://img.shields.io/github/issues-closed/infrahq/infra?color=green)](https://github.com/infrahq/infra/issues) [![GitHub commit activity](https://img.shields.io/github/commit-activity/m/infrahq/infra)](https://github.com/infrahq/infra/commits/main) [![YouTube Channel Views](https://img.shields.io/youtube/channel/views/UCft1MzQs2BJdW8BIUu6WJkw?style=social)](https://www.youtube.com/channel/UCft1MzQs2BJdW8BIUu6WJkw) [![GitHub Repo stars](https://img.shields.io/github/stars/infrahq/infra?style=social)](https://github.com/infrahq/infra/stargazers) [![Twitter Follow](https://img.shields.io/twitter/follow/infrahq?style=social)](https://twitter.com/infrahq)

## Infra

Manage access to Kubernetes, SSH, Databases, AWS and more:

1. Connect your infrastructure
2. Add your team via your identity provider (Google, Okta or Azure AD), or Infra's built-in user management
3. Assign access (e.g. `view`, `edit` or `admin`) to any user or group
4. Users log in via Infra's CLI with `infra login`

That's it!

![dashboard](https://user-images.githubusercontent.com/251292/179054958-cba0e177-dd35-42ea-ad28-a6c8a79e697a.png)

## Get Started

Deploy Infra via `helm`:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm install infra infrahq/infra
```

### Connectors

| Connector          | Status      | Documentation                                          |
| ------------------ | ----------- | ------------------------------------------------------ |
| Kubernetes         | ✅ Stable   | [Link](https://infrahq.com/docs/connectors/kubernetes) |
| Postgres           | Coming soon |                                                        |
| SSH                | Coming soon |                                                        |
| AWS                | Coming soon |                                                        |
| Container Registry | Coming soon |                                                        |
| MongoDB            | Coming soon |                                                        |
| Snowflake          | Coming soon |                                                        |
| MySQL              | Coming soon |                                                        |
| RDP                | Coming soon |                                                        |

## Documentation

- [Log in via Infra CLI](https://infrahq.com/docs/configuration/logging-in)
- [What is Infra?](https://infrahq.com/docs/getting-started/what-is-infra)
- [Architecture](https://infrahq.com/docs/reference/architecture)
- [Security](https://infrahq.com/docs/reference/security)
- [Helm Chart Reference](https://infrahq.com/docs/reference/helm-reference)

## Community

- [Community Forum](https://github.com/infrahq/infra/discussions) Best for: help with building, discussion about infrastructure access best practices.
- [GitHub Issues](https://github.com/infrahq/infra/issues) Best for: bugs and errors you encounter using Infra.
