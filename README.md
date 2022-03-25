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
* Onboard and offboard users via Okta (Active Directory, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)

## Quickstart

**Prerequisites:**
* Install [Helm](https://helm.sh/) (v3+)
* Install [Kubernetes](https://kubernetes.io/) (v1.14+)


### 1. Install Infra

```
helm repo add infrahq https://helm.infrahq.com/
helm repo update
helm install infra infrahq/infra
```

### 2. Install Infra CLI 

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

### 3. Login to Infra

```
infra login localhost
```

This will output the Infra Access Key which you will use to login, please store this in a safe place as you will not see this again.


### 4. Connect the first Kubernetes cluster

```
infra destinations add kubernetes example-name
``` 

### 5. Create the first local user 

``` 
infra id add name@example.com 
```

### 6. Grant Infra administrator privileges to the first user

``` 
infra grants add -u name@example.com --role admin infra 
``` 

### 7. Grant Kubernetes cluster administrator privileges to the first user 
```
infra grants add -u name@example.com --role cluster-admin kubernetes.example-name
```

### 8. Login to Infra with the newly created user 

```
infra login 
``` 
Select the Infra instance, and login with username / password

### 9. Use your Kubernetes clusters

You can now access the connected Kubernetes clusters via your favorite tools directly. Infra in the background automatically synchronizes your Kubernetes configuration file (kubeconfig). 

Alternatively, you can switch Kubernetes contexts by using the `infra use` command: 

```
infra use cluster-name
```

## [Security](./docs/security.md)

We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com).

## [Documentation](./docs)

* [Infra CLI Reference](./docs/cli.md)
* [Helm Chart Reference](./docs/helm.md)
* [Contributing](./docs/contributing.md)
* [License](./LICENSE)
