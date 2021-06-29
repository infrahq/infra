### Connect a Kubernetes cluster

First, retrieve your default API Key

```
infra apikey list
```

Then, install Infra Engine on the cluster:

```bash
helm install infra-engine infrahq/engine --namespace infra --set registry=<REGISTRY HOST> --set apiKey=<API KEY> --set name=<CLUSTER NAME>
```

> Note: if using Docker Desktop or Minikube, use `--set registry=infra`.

Verify the cluster has been connected:

```
infra destination list
```

To switch to this cluster, run

```
kubectl config use-context <CLUSTER NAME>
```
