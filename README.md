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

## Quickstart

### Install Infra registry

**Prerequisites:**
* [Helm](https://helm.sh/)

```
helm repo add infrahq https://helm.infrahq.com
helm repo update

helm install infra-registry infrahq/registry --namespace infrahq --create-namespace
```

### Connect Kubernetes cluster to Infra Registry

Run the following commands to retrive Infra Registry information and its API Key:

```
export INFRA_REGISTRY=$(kubectl get svc -n infrahq infra-registry -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}")
export INFRA_API_KEY=$(kubectl get secrets/infra-registry --template={{.data.defaultApiKey}} --namespace infrahq | base64 -d)
```

Then, install Infra Engine in the Kubernetes context of the cluster you want to connect to Infra Registry
```
helm install infra-engine infrahq/engine \
    --namespace infrahq \
    --set registry=$INFRA_REGISTRY \
    --set apiKey=$INFRA_API_KEY \
    --set name=my-first-cluster
```

### Connect an identity provider

First, add Okta via an `infra.yaml` configuration file:

* [Okta configuration guide](./docs/okta.md)

Next, add the following to your `infra.yaml` configuration file to grant everyone view access to the cluster.

```
groups:
  - name: Everyone    # example group
    source: okta
    roles:
      - name: view
        kind: cluster-role
        destinations:
          - name: my-first-cluster
```

Then update your Infra Registry with this new config:

```
helm upgrade infra-registry infrahq/registry --set-file config=./infra.yaml -n infrahq
```

### Install Infra CLI
<details>
  <summary><strong>Debian-based Distributions (Debian, Ubuntu)</strong></summary>

  ```
  case $(uname -m) in x86_64) ARCH=amd64 ;; aarch64) ARCH=arm64 ;; esac
  wget https://releases.infrahq.com/infra/v0.2.2/infra_v0.2.2_$ARCH.deb
  dpkg -i infra_v0.2.2_$ARCH.deb
  ```
</details>

<details>
  <summary><strong>Red Hat-based Distributions (Fedora, RHEL)</strong></summary>

  ```
  wget https://releases.infrahq.com/infra/v0.2.2/infra-v0.2.2-$(uname -m).rpm
  yum install infra-v0.2.2-$(uname -m).rpm
  ```
</details>

<details>
  <summary><strong>Alpine Linux</strong></summary>

  ```
  wget https://releases.infrahq.com/infra/v0.2.2/infra-v0.2.2=$(uname -m).apk
  apk add --allow-untrusted infra-v0.2.2-$(uname -m).apk
  ```
</details>

<details>
  <summary><strong>macOS</strong></summary>

  ```
  brew install infrahq/tap/infra
  ```
</details>

<details>
  <summary><strong>Windows</strong></summary>

  ```
  scoop bucket add infrahq https://github.com/infrahq/scoop.git
  scoop install infra
  ```
</details>
<br />

### Accessing infrastructure

```
infra login <your infra registry endpoint>
```

After login, Infra will automatically synchronize all the Kubernetes clusters configured for the user into their default kubeconfig file.

That's it! You now have access to your cluster via Okta. To list all the clusters, run `infra list`.

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
