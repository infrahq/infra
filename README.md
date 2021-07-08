<p align="center">
  <img src="./docs/images/header.svg" width="838" />
</p>

## Introduction
Infra is **identity and access management** for Kubernetes. Provide any user fine-grained access to Kubernetes clusters via existing identity providers such as Okta, Google Accounts, Azure Active Directory and more.

**Features**:
* One-command access: `infra login`
* Fine-grained permissions
* Onboard & offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)
* CLI & REST API
* Configure via `infra.yaml`

<p align="center">
  <img width="838" src="./docs/images/arch.svg" />
</p>

## Quickstart

### Install Infra

```
helm repo add infrahq https://helm.infrahq.com
helm install infra infrahq/infra
```

Infra exposes an **external ip** via a load balanacer. To list services and check which IP is exposed, run:

```
kubectl get svc infra
```

### Install Infra CLI

**macOS & Linux**

```
brew install infrahq/tap/infra
```

**Windows**

```
scoop bucket add infrahq https://github.com/infrahq/scoop.git
scoop install infra
```

### Login

```
infra login <EXTERNAL-IP>
```

Great! You're now **logged into the cluster via Infra**. 

### Adding users
* [Connect Okta](./docs/okta.md)
* [Add users manually](./docs/users.md)

## Documentation
* [Connect another cluster](./docs/connect.md)
* [Add a custom domain](./docs/domain.md)
* [CLI Reference](./docs/cli.md)
* [Configuration Reference](./docs/configuration.md)
* [Contributing](./docs/contributing.md)

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
