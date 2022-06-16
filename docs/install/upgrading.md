---
position: 3
---

# Upgrading

## Checking the version

To check the current version of Infra, run `infra version`:

```
$ infra version

    Client: 0.13.3
    Server: 0.13.3
```

## Upgrading Infra CLI

{% tabs %}
{% tab label="macOS" %}
```
brew update
brew upgrade infra
```
{% /tab %}
{% tab label="Windows" %}
```powershell
scoop update infra
```
{% /tab %}
{% tab label="Linux" %}

#### Ubuntu & Debian

```
sudo apt update
sudo apt upgrade infra
```

#### Fedora & Red Hat Enterprise Linux

```
sudo dnf update infra
```
{% /tab %}
{% /tabs %}

## Upgrading Infra Server

1. Update the Helm repository

    ```
    helm repo update infrahq
    ```

2. Upgrade Infra via `helm upgrade`:

    ```
    helm upgrade infra infrahq/infra
    ```

## Upgrading Infra Connector

1. Update the Helm repository

    ```
    helm repo update infrahq
    ```

2. Upgrade Infra iva `helm upgrade`:

    ```
    helm upgrade infra-connector infrahq/infra
    ```
