# Destination / Kubernetes

## Configure Kubernetes Destination

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
  INFRA_HOST=$(kubectl -n infrahq get ingress -l infrahq.com/component=registry -o jsonpath="{.items[].status.loadBalancer.ingress[*]['ip', 'hostname']}")
  ```
</details>

<details>
  <summary><strong>LoadBalancer</strong></summary>

  Note: It may take a few minutes for the LoadBalancer endpoint to be assigned. You can watch the status of the service with:

  ```
  kubectl -n infrahq get services -l infrahq.com/component=registry -w
  ```

  ```
  INFRA_HOST=$(kubectl -n infrahq get services -l infrahq.com/component=registry -o jsonpath="{.items[].status.loadBalancer.ingress[*]['ip', 'hostname']}")
  ```
</details>

<details>
  <summary><strong>ClusterIP</strong></summary>

  ```
  CONTAINER_PORT=$(kubectl -n infrahq get services -l infrahq.com/component=registry -o jsonpath="{.items[].spec.ports[0].port}")
  kubectl -n infrahq port-forward service infra 8080:$CONTAINER_PORT &
  INFRA_HOST='localhost:8080'
  ```
</details>

### Get Infra API Key

```
INFRA_API_KEY=$(kubectl -n infrahq get secrets infra-engine -o jsonpath='{.data.engine-api-key}' | base64 --decode)
```

---

```
helm install -n infrahq --create-namespace --set registry=$INFRA_HOST --set apiKey=$INFRA_API_KEY engine infrahq/engine
```

See [Helm Chart reference](./helm.md) for a complete list of options configurable through Helm.
