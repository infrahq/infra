<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/251292/179056556-361358af-aab9-4096-a714-87184f1afb22.svg">
    <img alt="logo" src="https://user-images.githubusercontent.com/251292/179056708-48e3c20b-22d1-4a40-9860-2f120c52a34f.svg">
  </picture>
</p>

![GitHub commit checks state](https://img.shields.io/github/checks-status/infrahq/infra/main?label=Build) ![GitHub closed issues](https://img.shields.io/github/issues-closed/infrahq/infra?color=green) ![GitHub commit activity](https://img.shields.io/github/commit-activity/m/infrahq/infra) ![YouTube Channel Views](https://img.shields.io/youtube/channel/views/UCft1MzQs2BJdW8BIUu6WJkw?style=social) ![GitHub Repo stars](https://img.shields.io/github/stars/infrahq/infra?style=social) ![Twitter Follow](https://img.shields.io/twitter/follow/infrahq?style=social)

## Infra

**Open-source access management** for Kubernetes, SSH, Databases, AWS and more:

1. Connect your infrastructure
2. Add your identity provider (Google, Okta or Azure AD), or use Infra's built-in users & groups
3. Assign access (e.g. `view`, `edit` or `admin`) to users & groups on your team
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
