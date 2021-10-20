<p align="center">
  <img src="./docs/images/InfraGithub.png"/>
</p>

## Introduction

Infra is **identity and access management** for Kubernetes. Provide any user fine-grained access to Kubernetes clusters via existing identity providers such as Okta, Google Accounts, Azure Active Directory and more.

**Features**:
* Single-command access: `infra login`
* No more out-of-sync `kubeconfig` files
* Fine-grained role assignment
* Onboard and offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)

## Contents

* [Introduction](#introduction)
* [Contents](#contents)
* [Quickstart](#quickstart)
  * [Create your first Infra configuration](#create-your-first-infra-configuration)
    * [Example](#example)
  * [Install Infra](#install-infra)
  * [Install Infra CLI](#install-infra-cli)
  * [Access your infrastructure](#access-your-infrastructure)
* [Next Steps](#next-steps)
  * [Connect additional identity providers](#connect-additional-identity-providers)
  * [Connect additional infrastructure](#connect-additional-infrastructure)
  * [Updating Infra](#updating-infra)
* [Contributing](#contributing)
* [Security](#security)
* [License](#license)
* [Documentation](#documentation)
  * [Identity Sources](#identity-sources)
  * [Infrastructure Destinations](#infrastructure-destinations)

## Quickstart

**Prerequisites:**
* [Helm](https://helm.sh/)

### Create your first Infra configuration

This example uses Okta and grants the "Everyone" group read-only access to the default namespace. See [Okta](./docs/sources/okta.md) for detailed Okta configuration steps.

#### Example

```yaml
# infra.yaml
---
registry:
  config:
    sources:
      - kind: okta
        domain: <Okta domain>
        clientId: <Okta client ID>
        clientSecret: <Okta client secret>
        apiToken: <Okta API token>
    groups:
      - name: Everyone
        roles:
            - kind: role
              name: viewer
              destinations:
                - name: <cluster name>
                  namespace: default
```

### Install Infra

```bash
helm repo add infrahq https://helm.infrahq.com/
helm repo update
helm install -f infra.yaml infra infrahq/infra
```

[Helm Chart Reference](./docs/helm.md)

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

### Access your infrastructure

```bash
infra login <registry endpoint>
```

Follow the instructions on screen to login.

<!--
TODO: add a login video
-->

[Infra CLI Reference](./docs/cli.md)

## Next Steps

### Updating Infra

```
helm repo update
helm upgrade -f infra.yaml infra infrahq.com/infra
```

## [Contributing](./docs/contributing.md)

## [Security](./docs/security.md)

We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com).

## [License](./LICENSE)

## [Documentation](./docs)

* [API Reference](./docs/api.md)
* [Infra CLI Reference](./docs/cli.md)
* [Helm Chart Reference](./docs/helm.md)

### [Identity Sources](./docs/sources)

* [Okta](./docs/sources/okta.md)

### [Infrastructure Destinations](./docs/destinations)

* [Kubernetes](./docs/destinations/kubernetes.md)
