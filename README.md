<p align="center">
  <img src="./docs/images/header.svg" width="838" />
</p>

## Introduction
Infra is **identity and access management** for Kubernetes. Provide any user fine-grained access to Kubernetes clusters via existing identity providers such as Okta, Google Accounts, Azure Active Directory and more.

**Features**:
* One-command access: `infra login`
* No more out of sync Kubeconfig files
* Fine-grained role assignment
* Onboard & offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)

<p align="center">
  <img width="838" src="./docs/images/arch.svg" />
</p>

## Quickstart

```
helm repo add infrahq https://helm.infrahq.com
helm install infra infrahq/infra
```

### Next steps 
* [Connect a Kubernetes cluster](./docs/connect.md)
* [Configure roles](./docs/permissions.md)
* [Access clusters via Infra CLI](./docs/access.md)
* [Add users via Okta integration](./docs/okta.md)
* [Add a custom domain](./docs/domain.md)

## Documentation
* [Helm Chart Reference](./docs/helm.md)
* [CLI Reference](./docs/cli.md)
* [Contributing](./docs/contributing.md)
* [Configuration reference](./docs/configuration.md)

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
