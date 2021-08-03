## Access clusters via Infra CLI

### Install Infra CLI

**macOS & Linux**

```
brew install infrahq/tap/infra
```

**Windows**

```
scoop bucket add infrahq https://github.com/infrahq/scoop.git
scoop install infra
```

### Login to your Infra Registry

```
infra login <your infra registry hostname>
```

### List clusters

```
infra list
```

### Switch to a Kubernetes context

```
kubectl config use-context <name>
```

Great! You've **logged into your cluster via Infra**. 
