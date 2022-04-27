# Upgrading

You can also download the [latest Infra release][1] directly from the repository.

## Upgrading Infra

1. Update the Helm repository

    ```bash
    helm repo update infrahq
    ```

2. Upgrade Infra. Ensure when upgrading a Helm chart to pass the same configuration values as during installation.

    ```bash
    helm upgrade [-f values.yaml] infra infrahq/infra
    ```

3. Wait for the pods to finish upgrade

    ```bash
    kubectl wait --for=condition=ready pod --selector app.kubernetes.io/name=infra-server
    ```

4. Check Infra version

    ```bash
    infra version
    ```

## Upgrading Infra Kubernetes Connector

1. Update the Helm repository

    ```bash
    helm repo update infrahq
    ```

2. Upgrade Infra. If using Helm values files, ensure those are passed into the upgrade command.

    ```bash
    helm upgrade -f values.yaml infra-connector infrahq/infra
    ```

    If using output from `infra destinations add`, ensure the same arguments are being passed into the upgrade command.

    ```bash
    helm upgrade --set connector.config.name=... --set connector.config.accessKey=... --set connector.config.server=... infra-connector infrahq/infra
    ```

3. Wait for the pods to finish upgrade

    ```bash
    kubectl wait --for=condition=ready pod --selector app.kubernetes.io/name=infra-connector
    ```

4. Check Infra Kubernetes Connector version

    ```bash
    kubectl logs -l app.kubernetes.io/name=infra-connector | grep 'starting infra'
    ```

## Upgrading Infra CLI

### macOS

1. Update Homebrew

    ```bash
    brew update
    ```

2. Upgrade Infra CLI

    ```bash
    brew upgrade infra
    ```

3. Check Infra CLI version

    ```bash
    infra version
    ```

### Windows

```powershell
scoop update infra
```

### Linux

```bash
# Ubuntu & Debian
sudo apt update
sudo apt upgrade infra
```

```bash
# Fedora & Red Hat Enterprise Linux
sudo dnf update infra
```

### Other Distributions

Binary releases can be downloaded and installed directly from the repository.

<details>
  <summary><strong>x86_64</strong></summary>

<!-- {x-release-please-start-version} -->
  ```bash
  LATEST=0.11.1
  curl -sSL https://github.com/infrahq/infra/releases/download/v$LATEST/infra_${LATEST}_linux_x86_64.zip
  unzip -d /usr/local/bin infra_${LATEST}_linux_x86_64.zip
  ```
<!-- {x-release-please-end} -->
</details>

<details>
  <summary><strong>ARM</strong></summary>

<!-- {x-release-please-start-version} -->
  ```bash
  LATEST=0.11.1
  curl -sSL https://github.com/infrahq/infra/releases/download/v$LATEST/infra_${LATEST}_linux_arm64.zip
  unzip -d /usr/local/bin infra_${LATEST}_linux_arm64.zip
  ```
<!-- {x-release-please-end} -->
</details>

[1]: https://github.com/infrahq/infra/releases/latest
