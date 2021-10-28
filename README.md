<p align="center">
  <img src="./docs/images/InfraGithub.png" width="806px" />
</p>

## Introduction

Infra is **identity and access management** for your cloud infrastructure. It puts the power of fine-grained access to infrastructure like Kubernetes in your hands via existing identity providers such as Okta, Google Accounts, Azure Active Directory and more.

**Features**:
* Single-command access: `infra login`
* No more out-of-sync user configurations
* Fine-grained role assignment
* Onboard and offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)

## Quickstart

**Prerequisites:**
* [Helm](https://helm.sh/) (v3+)
* [Kubernetes](https://kubernetes.io/) (v1.14+)

### Install Infra

```bash
helm repo add infrahq https://helm.infrahq.com/
helm repo update
helm install -n infrahq --create-namespace infra infrahq/infra
```

See [Helm Chart reference](./helm.md) for a complete list of options configurable through Helm.

### Configure Infra

This example configuration uses Okta and grants the "Everyone" group read-only access to the default namespace. You will need:

* Okta domain
* Okta client ID
* Okta client secret
* Okta API token
* Cluster name

See [Okta](./docs/providers/okta.md) for detailed Okta configuration steps.

Cluster name is auto-discovered or can be set statically in Helm with `engine.name`.

Also see [secrets.md](./docs/secrets.md) for details on how secrets work.

```yaml
# example values.yaml
---
config:
  providers:
    - kind: okta
      domain: <Okta domain>
      client-id: <Okta client ID>
      client-secret: <secret kind>:<Okta client secret name>
      okta:
        api-token: <secret kind>:<Okta API token name>
  groups:
    - name: Everyone
      roles:
          - kind: role
            name: viewer
            destinations:
              - name: <cluster name>
                namespaces:
                  - default
```

See the [Configuration reference](./docs/configuration.md) for a complete list of configurable options.

### Update Infra With Your Configuration

```
helm repo update
helm upgrade -n infrahq -f values.yaml infra infrahq/infra
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

### Access Your Infrastructure

First you need to get your Infra endpoint. This step may be different depending on your service type.

<details>
  <summary><strong>Ingress</strong></summary>

  ```
  INFRA_HOST=$(kubectl -n infrahq get ingress -l infrahq.com/component=infra -o jsonpath="{.items[].status.loadBalancer.ingress[*]['ip', 'hostname']}")
  ```
</details>

<details>
  <summary><strong>LoadBalancer</strong></summary>

  Note: It may take a few minutes for the LoadBalancer endpoint to be assigned. You can watch the status of the service with:

  ```
  kubectl -n infrahq get services -l infrahq.com/component=infra -w
  ```

  ```
  INFRA_HOST=$(kubectl -n infrahq get services -l infrahq.com/component=infra -o jsonpath="{.items[].status.loadBalancer.ingress[*]['ip', 'hostname']}")
  ```
</details>

<details>
  <summary><strong>ClusterIP</strong></summary>

  ```
  CONTAINER_PORT=$(kubectl -n infrahq get services -l infrahq.com/component=infra -o jsonpath="{.items[].spec.ports[0].port}")
  kubectl -n infrahq port-forward services infra 8080:$CONTAINER_PORT &
  INFRA_HOST='localhost:8080'
  ```
</details>

Once you have your infra host, it is time to login.

```bash
infra login $INFRA_HOST
```

Follow the instructions on screen to complete the login process.

See the [Infra CLI reference](./docs/cli.md) for more ways to use `infra`.

## Next Steps

### Connect Additional Identity Providers

* [Providers](./docs/providers)
  * [Okta](./docs/providers/okta.md)

### Connect Additional Infrastructure Destinations

* [Destinations](./docs/destinations)
  * [Kubernetes](./docs/destinations/kubernetes.md)

### Upgrade Infra

```
helm repo update
helm upgrade -n infrahq -f values.yaml infra infrahq/infra
```

## [Security](./docs/security.md)

We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com).

## [Documentation](./docs)

* [API Reference](./docs/api.md)
* [Infra CLI Reference](./docs/cli.md)
* [Helm Chart Reference](./docs/helm.md)
* [Contributing](./docs/contributing.md)
* [License](./LICENSE)
