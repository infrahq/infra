# Kubernetes

## Connect

Add the helm Infra repository:

```
helm repo add infrahq https://infrahq.github.io/helm-charts
helm repo update
```

Next, create an access key using the `infra` CLI:

```
INFRA_ACCESS_KEY=$(infra keys add --connector -q)
```

Lastly, deploy Infra on the Kubernetes cluster:

```
helm install infra infrahq/infra --set config.name=example --set config.accessKey=$INFRA_ACCESS_KEY
```

> To configure how the Infra connector is deployed, modify the [Helm values file](https://github.com/infrahq/helm-charts/blob/main/charts/infra/values.yaml).

## Authentication

Infra automatically generates the current user's Kubernetes Kubeconfig for all the connected clusters when running `infra login`:

```bash
infra login
```

`infra login` also respects the KUBECONFIG variable.

```bash
KUBECONFIG=~/.kube/custom-config infra login
```

### Switching Kubernetes clusters

Infra supports Kubernetes natively, and all existing tools that work with Kubernetes will continue to work.

Run `kubectl` to switch to a connected Kubernetes cluster:

```
kubectl config use-context example
```

Lastly, run a command against the cluster:

```
kubectl get pods -A
```

## Access control

To grant access, run `infra grant`:

```bash
infra grants add --group Engineering my-cluster --role cluster-admin
```

### Namespaces

Use Infra's resource notation to grant access to a namespace in the format:

```
<cluster>.<namespace>
```

For example, to grant `view` access to the `kube-system` namespace:

```bash
infra grants add --group Engineering my-cluster.kube-system --role view
```

### Roles

| Role            | Description                                                                                                                                                      |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cluster-admin` | Access to any resource                                                                                                                                           |
| `admin`         | Access to most resources, including roles and role bindings, but does not grant access to cluster-level resources such as cluster roles or cluster role bindings |
| `edit`          | Access to most resources in the namespace but does not grant access to roles or role bindings                                                                    |
| `view`          | Access to read most resources in the namespace but does not grant write access nor does it grant read access to secrets                                          |
| `logs`          | Access to pod logs                                                                                                                                               |
| `exec`          | Access to `kubectl exec`                                                                                                                                         |
| `port-forward`  | Access to `kubectl port-forward`                                                                                                                                 |

**Custom Kubernetes Roles**

If the provided roles are not sufficient, additional roles can be configured to integrate with Infra. To add a new role, create a ClusterRole in a connected cluster with label `app.infrahq.com/include-role=true`.

```bash
kubectl create clusterrole example --verb=get --resource=pods
kubectl label clusterrole/example app.infrahq.com/include-role=true
```
