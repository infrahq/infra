---
title: Upgrade Infra
position: 4
---

# Upgrade

## Checking the version

To check the current version of Infra, run `infra version`:

```shell
$ infra version

    Client: 0.15.2
```

## Upgrade Infra CLI

{% tabs %}
{% tab label="macOS" %}

```bash
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

```bash
sudo dpkg -i infra_*.deb
```

```bash
sudo apt install ./infra_*.deb
```

{% /tab %}
{% tab label="Fedora & RHEL" %}
Download the [latest][1] RPM package from GitHub and install it with `rpm` or `dnf`.

```bash
sudo rpm -U infra-*.rpm
```

```bash
sudo dnf install infra-*.rpm
```

{% /tab %}
{% tab label="Manual" %}
Download the [latest][1] release from GitHub, unpack the file, and add the binary to the `PATH`.
{% /tab %}
{% /tabs %}

## Upgrade Infra Connector

To update your connector to the latest version, ensure you have added the Helm repo and updated it:

```bash
helm repo add infrahq https://helm.infrahq.com
helm repo update
```

Then, referring to the `values.yaml` file you created at initial install, run the Helm update command:

```bash
helm upgrade --install infra infrahq/infra --values values.yaml
```

[1]: https://github.com/infrahq/infra/releases/latest
