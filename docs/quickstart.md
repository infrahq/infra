# Quickstart

## Deploy Infra

To deploy Infra, follow the [deployment guide](./deploy.md).

## Connect via the CLI

Download the Infra CLI:

### macOS

Download via [homebrew](https://brew.sh):

```
brew install infrahq/tap/infra
```

### Windows

Download via [scoop](https://scoop.sh):

```powershell
scoop bucket add infrahq https://github.com/infrahq/scoop.git
scoop install infra
```

### Ubuntu & Debian

Download the [latest](https://github.com/infrahq/infra/releases/latest) Debian package from GitHub and install it with `dpkg` or `apt`.

```
sudo dpkg -i infra_*.deb
```

```
sudo apt install ./infra_*.deb
```

### Fedora & RHEL

Download the [latest](https://github.com/infrahq/infra/releases/latest) RPM package from GitHub and install it with `rpm` or `dnf`.

```
sudo rpm -i infra-*.rpm
```

```
sudo dnf install infra-*.rpm
```

### Manual

Download the [latest](https://github.com/infrahq/infra/releases/latest) release from GitHub, unpack the file, and add the binary to the `PATH`.

Next, log in:

```
infra login <your infra host>
```

Then, create an access key:

```
INFRA_ACCESS_KEY=$(infra keys add --connector -q)
```

## Access your cluster

Grant yourself access to the cluster:

```
infra grants add <your user email> example --role view
```

Next, verify your access:

```
infra list
```

Then, run `kubectl` to switch to your newly connected cluster.

```
kubectl config use-context infra:example
```

Alternatively, you can switch clusters via `infra use` command.

```
infra use example
```

Lastly, try running a command on the Kubernetes cluster:

```
kubectl get pods -A
```

> By default, Infra will give view access to the user who made the install. To modify permissions or give additional access, use Infra dashboard or [CLI](integrations/kubernetes#access-control).

## Next steps

Congratulations. You've successfully connected your first cluster.

Infra works best when used with a team. Next, configure how users authenticate by connecting an [identity provider](./manage/authentication.md#identity-providers), or add users directly by [inviting them](./manage/users-groups#adding-a-user).
