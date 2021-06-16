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

### Install Infra Registry

```
kubectl apply -f https://raw.githubusercontent.com/infrahq/early-access/main/deploy/registry.yaml
```

Infra exposes a `LoadBalancer` service by default. Find the **External IP** of the load balancer:

```
kubectl get svc --namespace infra
```

### Install Infra CLI

```
curl -L "https://github.com/infrahq/early-access/releases/latest/download/infra-$(uname -s)-$(uname -m)" -o /usr/local/bin/infra && chmod +x /usr/local/bin/infra
```

### Log in

```
infra login <EXTERNAL-IP>
```

### Connect a Kubernetes cluster

```
$ infra connect kubernetes --name cluster_name 

To add a Kubernetes cluster to Infra, run the following command 

helm install infrahq/infra --set infra.apiKey=120d8j102d8j102d8j1028d --set infra.server=<Pre-filled> --set infra.name="my-first-cluster" 

```

## Documentation
* [Connect Okta](./docs/okta.md)
* [Add users manually](./docs/users.md)
* [Add a custom domain](./docs/domain.md)
* [CLI Reference](./docs/cli.md)
* [Configuration Reference](./docs/configuration.md)
* [Contributing](./docs/contributing.md)

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
