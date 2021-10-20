<p align="center">
  <img src="./docs/images/InfraGithu2.svg" width="851" />
</p>

## Introduction

Infra is **identity and access management** for Kubernetes. Provide any user fine-grained access to Kubernetes clusters via existing identity providers such as Okta, Google Accounts, Azure Active Directory and more.

**Features**:
* Single-command access: `infra login`
* No more out-of-sync kubeconfig files
* Fine-grained role assignment
* Onboard and offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)

## Quickstart

### Install Infra Registry

**Prerequisites:**
* [Helm](https://helm.sh/)

```bash
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm install infra-registry infrahq/registry --namespace infrahq --create-namespace
```

### Connect Kubernetes cluster to Infra Registry

Once the load balancer for the Infra Registry is available, run the following commands to retrieve Infra Registry information and its engine API key:

```bash
INFRA_REGISTRY=$(kubectl --namespace infrahq get services infra-registry -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}")
ENGINE_API_KEY=$(kubectl --namespace infrahq get secrets infra-registry -o jsonpath='{.data.engineApiKey}' | base64 -d)
```

Then, install Infra Engine in the Kubernetes context of the cluster you want to connect to Infra Registry:

```bash
helm install infra-engine infrahq/engine --namespace infrahq --set registry=$INFRA_REGISTRY --set apiKey=$ENGINE_API_KEY
```

### Connect an identity provider

First, add Okta via an `infra.yaml` configuration file:

* [Okta configuration guide](./docs/okta.md)

Next, add the following to your `infra.yaml` configuration file to grant everyone view access to the cluster:

```yaml
groups:
  - name: Everyone    # example group
    source: okta
    roles:
      - name: view
        kind: cluster-role
        destinations:
          - name: <cluster name>
```

Then, update your Infra Registry with this new config:

```bash
helm upgrade infra-registry infrahq/registry --namespace infrahq --set-file config=./infra.yaml
```

### Install Infra CLI
<details>
  <summary><strong>Debian, Ubuntu</strong></summary>

  ```bash
  sudo echo 'deb [trusted=yes] https://apt.fury.io/infrahq/ /' >/etc/apt/sources.list.d/infrahq.list
  sudo apt update
  sudo apt install infra
  ```
</details>

<details>
  <summary><strong>Fedora, Red Hat Enterprise Linux</strong></summary>

  ```bash
  sudo dnf config-manager --add-repo https://yum.fury.io/infrahq/
  sudo dnf install infra
  ```
</details>

<details>
  <summary><strong>macOS</strong></summary>

  ```bash
  brew install infrahq/tap/infra
  ```
</details>

<details>
  <summary><strong>Windows</strong></summary>

  ```powershell
  scoop bucket add infrahq https://github.com/infrahq/scoop.git
  scoop install infra
  ```
</details>

### Access infrastructure

```bash
infra login <your infra registry endpoint>
```

After login, Infra will automatically synchronize all the Kubernetes clusters configured for the user into their default kubeconfig file.

That's it! You now have access to your cluster via Okta. To list all the clusters, run `infra list`.

## Upgrading Infra

First, update the Helm repo:

```bash
helm repo update
```

Then, update the Infra Registry

```bash
helm upgrade infra-registry infrahq/registry --namespace infrahq
```

Lastly, update any Infra Engines:

```bash
helm upgrade infra-engine infrahq/engine --namespace infrahq
```

## Next Steps
* [Update roles](./docs/permissions.md)
* [Add a custom domain](./docs/domain.md) to make it easy for sharing with your team
* [Connect more Kubernetes clusters](./docs/connect.md)

## Documentation
* [Okta Reference](./docs/okta.md)
* [Helm Chart Reference](./docs/helm.md)
* [CLI Reference](./docs/cli.md)
* [Contributing](./docs/contributing.md)

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
