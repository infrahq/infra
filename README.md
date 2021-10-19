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
  * [Identity Providers](#identity-providers)
  * [Infrastructure](#infrastructure)

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
                - name: my-first-cluster
                  namespace: default

engine:
  name: my-first-cluster
  
```

### Install Infra

[![helm](https://img.shields.io/badge/docs-helm-green?logo=bookstack&style=flat)](./docs/helm.md)

```bash
helm install --repo https://helm.infrahq.com/ --values infra.yaml infra infra
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

### Access your infrastructure

[![cli](https://img.shields.io/badge/docs-cli-green?logo=bookstack&style=flat)](./docs/cli.md)

```bash
infra login <registry endpoint>
```

Follow the instructions on screen to login.

<!--
TODO: add a login video
-->

## Next Steps

### Connect additional identity providers

[![sources](https://img.shields.io/badge/docs-sources-green?logo=bookstack&style=flat)](./docs/sources)

* [Connect Okta](./docs/sources/okta.md#connect)
<!--
* [Connect GitHub](./docs/sources/github.md#connect)
* [Connect Google](./docs/sources/google.md#connect)
* [Connect Azure AD](./docs/sources/azure-ad.md#connect)
* [Connect GitLab](./docs/sources/gitlab.md#connect)
-->

### Connect additional infrastructure

[![destinations](https://img.shields.io/badge/docs-destinations-green?logo=bookstack&style=flat)](./docs/destinations)

* [Connect Kubernetes Cluster](./docs/destinations/kubernetes.md#connect)

<!--
**Databases**
* [Connect PostgresQL](./docs/destinations/postgresql.md)
-->

<!--
**SSH**
* [Connect Secure Shell (SSH)](./docs/destinations/ssh.md)
-->

<!--
Publi Cloud
* [Connect Amazon Web Services (AWS)](./docs/destinations/aws.md)
* [Connect Google Cloud Platform (GCP)](./docs/destinations/gcp.md)
-->

### Updating Infra

```
helm upgrade --namespace infrahq --create-namespace --repo https://helm.infrahq.com/ --values infra.yaml infra infra
```

## Contributing

[![contributing](https://img.shields.io/badge/docs-contributing-green?style=flat)](./docs/contributing.md)
[![issues](https://img.shields.io/github/issues/infrahq/infra?style=flat)](https://github.com/infrahq/infra/issues)
[![pulls](https://img.shields.io/github/issues-pr/infrahq/infra?style=flat)](https://github.com/infrahq/infra/pulls)

## Security

We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com).

## License

[![license](https://img.shields.io/badge/license-apache-blue?style=flat)](./LICENSE)

## Documentation

[![docs](https://img.shields.io/badge/docs-apache-green?style=flat)](./docs)

* [API Reference](./docs/api.md)
* [Infra CLI Reference](./docs/cli.md)
* [Helm Chart Reference](./docs/helm.md)

### Identity Providers

* [Okta](./docs/sources/okta.md)
<!--
* [GitHub](./docs/sources/github.md)
* [Google](./docs/sources/google.md)
* [Azure AD](./docs/sources/azure-ad.md)
-->

### Infrastructure

* [Kubernetes](./docs/destinations/kubernetes.md)
<!--
* [PostgresQL](./docs/destinations/postgresql.md)
* [SSH](./docs/destinations/ssh.md)
* [AWS](./docs/destinations/aws.md)
* [GCP](./docs/destinations/gcp.md)
-->
