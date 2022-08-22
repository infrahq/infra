---
position: 3
---

# Upgrade

## Checking the version

To check the current version of Infra, run `infra version`:

```
$ infra version

    Client: 0.13.3
    Server: 0.13.3
```

## Upgrade Infra CLI

{% tabs %}
{% tab label="macOS" %}
```
brew upgrade infra
```
{% /tab %}
{% tab label="Windows" %}
```powershell
scoop update infra
```
{% /tab %}

{% tab label="Ubuntu & Debian" %}
Download the [latest][1] Debian package from GitHub and install it with `dpkg` or `apt`.
```
sudo dpkg -i infra_*.deb
```
```
sudo apt install ./infra_*.deb
```
{% /tab %}
{% tab label="Fedora & RHEL" %}
Download the [latest][1] RPM package from GitHub and install it with `rpm` or `dnf`.
```
sudo rpm -U infra-*.rpm
```
```
sudo dnf install infra-*.rpm
```
{% /tab %}
{% tab label="Manual" %}
Download the [latest][1] release from GitHub, unpack the file, and add the binary to the `PATH`.
{% /tab %}
{% /tabs %}

[1]: https://github.com/infrahq/infra/releases/latest

## Upgrade Infra Server

1. Update the Helm repository

    ```
    helm repo update infrahq
    ```

2. Upgrade Infra via `helm upgrade`:

    ```
    helm upgrade infra infrahq/infra
    ```

## Upgrade Infra Connector

1. Update the Helm repository

    ```
    helm repo update infrahq
    ```

2. Upgrade Infra iva `helm upgrade`:

    ```
    helm upgrade infra-connector infrahq/infra
    ```