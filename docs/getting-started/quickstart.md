# Quickstart

### Prerequisites:

- [Helm](https://helm.sh/) (v3+)
- [Kubernetes](https://kubernetes.io/) (v1.14+ – ie. Docker Desktop with Kubernetes)

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
brew link infrahq/tap/infra
```

</details>

<details>
  <summary><strong>Windows</strong></summary>

```bash
scoop bucket add infrahq https://github.com/infrahq/scoop.git
scoop install infra
```

</details>

<details>
  <summary><strong>Linux</strong></summary>

```bash
# Ubuntu & Debian
echo 'deb [trusted=yes] https://apt.fury.io/infrahq/ /' | sudo tee /etc/apt/sources.list.d/infrahq.list
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

This will output the Infra access key which you will use to login, please store this in a safe place as you will not see this again.

### 4. Connect the first Kubernetes cluster

In order to add connectors to Infra, you will need to generate an access key.

> Using the Infra access key from 3 is _not_ recommended as it provides more privileges than is necessary for a connector and may pose a security risk.

```bash
infra keys add <keyName> connector
```

Once you have a connector access key, install Infra into your Kubernetes cluster.

```bash
helm upgrade --install infra-connector infrahq/infra --set connector.config.name=<clusterName> --set connector.config.server=<serverAddress> --set connector.config.accessKey=<accessKey>
```

> If the connector will live in the same cluster and namespace as the server, you can set `connector.config.server=localhost`.

> You may also need to set `connector.config.skipTLSVerify=true` if the server is using a self-signed certificate.

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
