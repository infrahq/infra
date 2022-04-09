# Install Infra CLI

## macOS

```bash
brew install infrahq/tap/infra
```

You may need to perform `brew link` if your symlinks are not working.
```bash
brew link infrahq/tap/infra
```

## Windows

```powershell
scoop bucket add infrahq https://github.com/infrahq/scoop.git
scoop install infra
```

## Linux

### Ubuntu & Debian

```bash
echo 'deb [trusted=yes] https://apt.fury.io/infrahq/ /' | sudo tee /etc/apt/sources.list.d/infrahq.list
sudo apt update
sudo apt install infra
```

### Fedora & Red Hat Enterprise Linux
```bash
sudo dnf config-manager --add-repo https://yum.fury.io/infrahq/
sudo dnf install infra
```

## Upgrading

See [Upgrading Infra CLI](../operator-guides/upgrading-infra.md#upgrading-infra-cli)