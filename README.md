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

### Deploy Infra

```
kubectl apply -f https://raw.githubusercontent.com/infrahq/early-access/main/deploy/server.yaml
```

Infra exposes a `LoadBalancer` service by default. Find the **External IP** of the load balancer:

```
kubectl get svc --namespace infra
```

### Create admin user

```
kubectl -n infra exec deploy/infra -- infra users create admin@example.com passw0rd
kubectl -n infra exec deploy/infra -- infra grant admin@example.com infra --role infra.owner
```

### Install Infra CLI

```
curl -L "https://github.com/infrahq/early-access/releases/latest/download/infra-$(uname -s)-$(uname -m)" -o /usr/local/bin/infra && chmod +x /usr/local/bin/infra
```

### Log in

```
infra login -k -u admin@example.com -p passw0rd <EXTERNAL-IP>
```

### List users

```
infra users ls
```

### Connect a Kubernetes cluster

```
infra add example-cluster
```

### Verify cluster is connected

```
infra list
```

### Grant yourself access

```
infra grant admin@example.com example-cluster
```

### Connect to the cluster

```bash
# Switch to cluster
kubectl config use-context example-cluster

# List pods
kubectl get pods -A
```

You're now connected to this new cluster via Infra.

## Documentation
* [Add a custom domain](./docs/domain.md)
* [Manage Users](./docs/users.md)
* [Grant & revoke access via roles](./docs/access.md)
* [Connect Okta](./docs/okta.md)
* [CLI Reference](./docs/cli.md)
* [Configuration Reference](./docs/configuration.md)
* [Contributing](./docs/contributing.md)

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
