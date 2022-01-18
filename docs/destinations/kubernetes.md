# Kubernetes

## Switch Between Infra-Managed Kubernetes Clusters

```bash
$ infra k|k8s|kubernetes use|use-context [CLUSTER_NAME|CLUSTER_ID] [-l CLUSTER_LABEL[,CLUSTER_LABEL]...] [-n NAMESPACE] [-r ROLE]
```

### Examples

If multiple Infra-managed Kubernetes contexts exist, `infra kubernetes use-context` will display a prompt allowing you to select the desired cluster interactively. In brackets are the labels associated with each context.

```bash
$ infra kubernetes use-context
? Multiple candidates found:  [Use arrows to move, type to filter]
> b99617096a1e docker-desktop [local]
  825957601c8e infrahq-production [eks, us-east-1]
```

If multiple Infra-managed Kubernetes contexts exist with refined namespace access, `infra kubernetes use-context` will display a prompt allowing you to select the desired namespace once a cluster has been determined.

```bash
$ infra k use
? Multiple candidates found: b99617096a1e docker-desktop []
? Multiple candidates found:  [Use arrows to move, type to filter]
> * [edit, admin]
  kube-public [view]
  kube-system [view]
```

You can select a cluster by name...

```bash
$ infra kubernetes use-context docker-desktop
? Multiple candidates found:  [Use arrows to move, type to filter]
> * [edit, admin]
  kube-public [view]
  kube-system [view]
```

Or by ID...

```bash
$ infra kubernetes use-context b99617096a1e
? Multiple candidates found:  [Use arrows to move, type to filter]
> * [edit, admin]
  kube-public [view]
  kube-system [view]
```

Or with one or more of its labels...

```bash
$ infra kubernetes use-context -l local
? Multiple candidates found:  [Use arrows to move, type to filter]
> * [edit, admin]
  kube-public [view]
  kube-system [view]
```

You can select a namespace by name...

```bash
$ infra kubernetes use-context -l local -n kube-system
Switched to context "infra:docker-desktop:kube-system".
```

Or by role...

```bash
$ infra kubernetes use-context docker-desktop -r admin
Switched to context "infra:docker-desktop".
```

Any of these options can be used in combination to refine your search for the corrent context.


See the [Infra CLI reference](./docs/cli.md) for more.

## Configure Destination

### `destinations`

| Parameter      | Description                                      | Default               |
|----------------|--------------------------------------------------|-----------------------|
| `namespaces`   | Limit access to only these Kubernetes namespaces | `[]` (all namespaces) |

## Connect a Kubernetes Cluster

Before installing Infra in additional clusters, you will first need to gather some information about your main Infra deployment.

### Get Infra Endpoint

Depending on your Infra Helm configurations, the steps will differ.

<details>
  <summary><strong>Ingress</strong></summary>

  ```
  INFRA_SERVER=$(kubectl -n infrahq get ingress -l infrahq.com/component=infra -o jsonpath="{.items[].status.loadBalancer.ingress[*]['ip', 'hostname']}")
  ```
</details>

<details>
  <summary><strong>LoadBalancer</strong></summary>

  Note: It may take a few minutes for the LoadBalancer endpoint to be assigned. You can watch the status of the service with:

  ```
  kubectl -n infrahq get services -l infrahq.com/component=infra -w
  ```

  ```
  INFRA_SERVER=$(kubectl -n infrahq get services -l infrahq.com/component=infra -o jsonpath="{.items[].status.loadBalancer.ingress[*]['ip', 'hostname']}")
  ```
</details>

<details>
  <summary><strong>ClusterIP</strong></summary>

  ```
  CONTAINER_PORT=$(kubectl -n infrahq get services -l infrahq.com/component=infra -o jsonpath="{.items[].spec.ports[0].port}")
  kubectl -n infrahq port-forward service infra 8080:$CONTAINER_PORT &
  INFRA_SERVER='localhost:8080'
  ```
</details>

### Get Infra API Key

```
INFRA_API_TOKEN=$(kubectl -n infrahq get secrets infra-engine -o jsonpath='{.data.engine-api-token}' | base64 --decode)
```

---

```
helm upgrade --install -n infrahq --create-namespace --set config.server=$INFRA_SERVER --set config.apiToken=$INFRA_API_TOKEN infra infrahq/engine
```

See [Helm Chart reference](./helm.md) for a complete list of options configurable through Helm.
