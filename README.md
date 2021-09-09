<p align="center">
  <img src="./docs/images/header.svg" width="838" />
</p>

## Introduction
Infra is secure Kubernetes access for your team.

**Features**:
* One-command access: `infra login`
* No more out of sync Kubeconfig files
* Fine-grained role assignment
* Onboard & offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)

## Quickstart

### Install Infra

**Prerequisites:**
* [Helm](https://helm.sh/)

```
helm repo add infrahq https://helm.infrahq.com
helm repo update

helm install infra-registry infrahq/registry --namespace infrahq --create-namespace
```

### Connect Kubernetes

```
helm install infra-engine infrahq/engine --set registry=$INFRA_REGISTRY --set apiKey=$INFRA_API_KEY
```

### Connect Okta

* [See the Okta configuration guide](./docs/okta.md)

### Log in via Okta

**macOS & Linux**

```
brew install infrahq/tap/infra
```

**Windows**

```
scoop bucket add infrahq https://github.com/infrahq/scoop.git
scoop install infra
```

```
infra login <your infra registry endpoint>
```

After login, Infra will automatically synchronize all the Kubernetes clusters configured for the user into their default kubeconfig file. 

### Scoping permissions

To scope permissions access

### Accessing clusters 

To list all the clusters, please run `infra list`. 

Users can then switch Kubernetes context via `kubectl config use-context <name>` or via any Kubernetes tools. 

## Next Steps 
* [Add a custom domain](./docs/domain.md) to make it easy for sharing with your team 
* [Connect more Kubernetes clusters](./docs/connect.md)
* [Update roles](./docs/permissions.md) 

## Documentation
* [Okta Reference](./docs/okta.md)
* [Helm Chart Reference](./docs/helm.md)
* [CLI Reference](./docs/cli.md)
* [Contributing](./docs/contributing.md)
* [Configuration reference](./docs/configuration.md)

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
