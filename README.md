<p align="center">
  <img src="./docs/images/InfraGithub.png" />
</p>

**We take security very seriously.** If you believe you have found a security issue please report it to our security team by contacting us at security@infrahq.com.

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
* Install [Helm](https://helm.sh/) (v3+)
* Install [Kubernetes](https://kubernetes.io/) (v1.14+)

### Configure

#### Configure Okta

Follow the [Okta guide](./docs/providers/okta.md) to set up Okta for Infra. You'll need:

* Okta domain
* Okta client ID
* Okta client secret
* Okta API token

#### Configure Infra

```yaml
# example values.yaml
---
config:
  providers:
    - kind: okta
      # Update with values from above
      # Values can be securely loaded from different secret managers (e.g. Kubernetes secrets)
      # or in plaintext (not recommended for production). See https://github.com/infrahq/infra/blob/main/docs/secrets.md
      domain: <Okta domain>
      clientID: <Okta client id>
      clientSecret: <Okta client secret>
      apiToken: <Okta api token>

  groups:
    # Grants the "Everyone" Okta group read-only access
    # to the default namespace of your Kubernetes cluster
    - name: Everyone
      provider: okta
      roles:
        - kind: cluster-role
          name: view
          destinations:
            - name: <cluster name> # cluster name in your cloud provider
              namespaces:
                - default
```

See the [Helm Chart reference](./docs/helm.md) for a complete list of options configurable through Helm.

> Note: Infra uses [Secrets](https://github.com/infrahq/infra/blob/main/docs/secrets.md) to securely load secrets.
> It is _not_ recommended to use plain text secrets. Considering using another supported secret type.

### Install Infra

```bash
helm repo add infrahq https://helm.infrahq.com/
helm repo update
helm upgrade --install -n infrahq --create-namespace -f values.yaml infra infrahq/infra
```

### Install Infra CLI

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

<details>
  <summary><strong>Linux</strong></summary>
  
  ```bash
  # Ubuntu & Debian
  sudo echo 'deb [trusted=yes] https://apt.fury.io/infrahq/ /' >/etc/apt/sources.list.d/infrahq.list
  sudo apt update
  sudo apt install infra
  ```
  
  ```bash
  # Fedora & Red Hat Enterprise Linux
  sudo dnf config-manager --add-repo https://yum.fury.io/infrahq/
  sudo dnf install infra
  ```
</details>


### Access Your Infrastructure

You will need to get your Infra endpoint. This step will be different depending on your Service type.

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
helm upgrade --install -n infrahq -f values.yaml infra infrahq/infra
```

## [Security](./docs/security.md)

We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com).

## [Documentation](./docs)

* [API Reference](./docs/api.md)
* [Infra CLI Reference](./docs/cli.md)
* [Helm Chart Reference](./docs/helm.md)
* [Contributing](./docs/contributing.md)
* [License](./LICENSE)
