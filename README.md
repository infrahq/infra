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

### Install Infra

```
helm repo add infrahq https://helm.infrahq.com
helm install infra infrahq/infra
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

### Login to Infra

```
infra login <EXTERNAL-IP>
```

### Switch Kubernetes context

```
kubectl config use-context default
```

Great! You're now **logged into the cluster via Infra**. 

### Next steps 
* Add users to Infra 
  * [Add users via Okta integration](./docs/okta.md)
  * [Add users manually](./docs/users.md)
* Connect another Kubernetes cluster 
  * [Connect another cluster](./docs/connect.md)
* Add a [custom domain](./docs/domain.md) to infra login for quick access 

## Documentation
* [Helm Chart](./docs/helm.md)
* [CLI Reference](./docs/cli.md)
* [Configuration Reference](./docs/configuration.md)
* [Contributing](./docs/contributing.md)

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
