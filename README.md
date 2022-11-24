<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/251292/179098559-75b53555-e389-40cc-b910-0e53521efad2.svg">
    <img alt="logo" src="https://user-images.githubusercontent.com/251292/179098561-eaa231c1-5757-40d7-9e5f-628e5d9c3e47.svg">
  </picture>
</div>

<div align="center">

[![YouTube Channel Views](https://img.shields.io/youtube/channel/views/UCft1MzQs2BJdW8BIUu6WJkw?style=social)](https://www.youtube.com/channel/UCft1MzQs2BJdW8BIUu6WJkw) [![GitHub Repo stars](https://img.shields.io/github/stars/infrahq/infra?style=social)](https://github.com/infrahq/infra/stargazers) [![Twitter Follow](https://img.shields.io/twitter/follow/infrahq?style=social)](https://twitter.com/infrahq)

</div>

Infra provides authentication and access management to servers, clusters, and databases.

## Getting Started

#### macOS

```
brew install infrahq/tap/infra
```

#### Windows

```powershell
scoop bucket add infrahq https://github.com/infrahq/scoop.git
scoop install infra
```

#### Linux

Download the [latest](https://github.com/infrahq/infra/releases/latest) packages from GitHub and install it with `dpkg`,  `apt`, `rpm`, or `dnf`

```
sudo dpkg -i infra_*.deb
```

```
sudo apt install ./infra_*.deb
```

```
sudo rpm -i infra-*.rpm
```

```
sudo dnf install infra-*.rpm
```

### Create an access key

Log into Infra. If you don't have a self-hosted Infra configured, you can sign up for a [free Infra instance](https://signup.infrahq.com) to get started.

```
infra login 
```

You'll be prompted for the Infra URL you created when you signed up. (e.g. `<org>.infrahq.com`).

Then, create an access key:

```
INFRA_ACCESS_KEY=$(infra keys add --connector -q)
```

### Connect Kubernetes cluster

Install Infra connector via [helm](https://helm.sh):

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm install infra infrahq/infra --set connector.config.accessKey $INFRA_ACCESS_KEY --set connector.config.name=example
```

### Access your cluster

Run `kubectl` to switch to your newly connected cluster.

```
kubectl config use-context infra:example
```

Lastly, connect to the cluster:

```
kubectl get pods -A
```

**Note**
By default, Infra will give view access to the user who made the install. To modify permissions or give additional access, use Infra dashboard or [CLI](integrations/kubernetes#access-control).

## Next steps

Congratulations. You've successfully connected your first cluster.

Infra works best when used with a team. Next, configure how users authenticate by connecting an [identity provider](./manage/authentication.md#identity-providers), or add users directly by [inviting them](./manage/users-groups#adding-a-user).

## Community

- [Community Forum](https://github.com/infrahq/infra/discussions) Best for: help with building, discussion about infrastructure access best practices.
- [GitHub Issues](https://github.com/infrahq/infra/issues) Best for: bugs and errors you encounter using Infra.
