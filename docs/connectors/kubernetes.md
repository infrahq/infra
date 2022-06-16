---
title: Kubernetes
position: 1
---

# Kubernetes

## Connecting a cluster

First, generate an access key:

```
infra keys add KEY_NAME connector
```

Next, use this access key to connect your cluster:

```
helm upgrade --install infra-connector infrahq/infra \
    --set connector.config.server=INFRA_SERVER_HOSTNAME \
    --set connector.config.accessKey=ACCESS_KEY \
    --set connector.config.name=example-cluster-name \
    --set connector.config.skipTLSVerify=true # only include if you have not yet configured certificates
```

## Managing access

Once you've connected a cluster, you can grant access via `infra grants add`:

```
# grant access to a user
infra grants add fisher@example.com example --role admin

# grant access to a group
infra grants add -g engineering example --role view
```

### Roles

Roles supported by a connector are defined only in the context of the connected cluster. Infra supports the following roles by default.

| Role | Access level |
| --- | --- |
| `cluster-admin` | Grants access to any resource |
| `admin` | Grants access to most resources, including roles and role bindings, but does not grant access to cluster-level resources such as cluster roles or cluster role bindings |
| `edit` | Grants access to most resources in the namespace but does not grant access to roles or role bindings
| `view` | Grants access to read most resources in the namespace but does not grant write access nor does it grant read access to secrets |
| `logs` | Grants access to pod logs |
| `exec` | Grants access to `kubectl exec` |
| `port-forward` | Grants access to `kubectl port-forward` |

### Example: Grant user `dev@example.com` the `view` role to a cluster

This command will grant the user `dev@example.com` read-only access into a cluster, giving that user the privileges to query Kubernetes resources but not modify any resources.

```
infra grants add dev@example.com cluster --role view
```

### Example: Grant user `ops@example.com` the `admin` role to a namespace

This command will grant the user `ops@example.com` admin access into a namespace, giving that user the privileges to create, update, and delete any resource so long as the resources they’re modifying exist in the namespace.

```
infra grants add ops@example.com cluster.namespace --role admin
```

### Example: Revoke from the user `ops@example.com` the `admin` role to a namespace

This command will remove the `admin` role, granted in the previous example, from `ops@example.com`.

```
infra grants remove ops@example.com cluster.namespace --role cluster-admin
```

### Custom Kubernetes Roles

If the provided roles are not sufficient, additional roles can be configured to integrate with Infra. To add a new role, create a ClusterRole in a connected cluster with label `app.infrahq.com/include-role=true`.

```
kubectl create clusterrole example --verb=get --resource=pods
kubectl label clusterrole/example app.infrahq.com/include-role=true
```

## Additional Information

- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
