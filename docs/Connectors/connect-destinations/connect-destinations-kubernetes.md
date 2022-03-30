# Kubernetes

## Install with Helm

Using Infra CLI:

1. Generate the helm install command via

```
infra destinations add kubernetes example-cluster-name
```

2. Run the output Helm command on the Kubernetes cluster to be added.

Example output:
```
helm upgrade --install infra-engine infrahq/infra --set engine.config.accessKey=2pVqDSdkTF.oSCEe6czoBWdgc6wRz0ywK8y --set engine.config.name=kubernetes.example-cluster --set engine.config.server=https://infra.acme.com
```

## Grants

Kubernetes Grants are built on top of Kubernetes RBAC which consists of `Role` and `ClusterRole` and `RoleBinding` and `ClusterRoleBinding`. For more detailed explanation of these concepts, checkout the official documentation:

[Using RBAC Authorization](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)

To summarize:

- `Role` is a set of permissions applied to a specific namespace
- `ClusterRole` is a set of permissions applied to the entire cluster
- `ClusterRole` can also be applied to namespaced resources within a specific namespace
- A `Role` can only be referenced in the namespace it is created in

There are several default user-facing `Role` and `ClusterRole` defined in most clusters and is generally sufficient for most use cases.

| Name | With Namespace | Without Namespace |
| --- | --- | --- |
| cluster-admin | Grants access to any resource in the namespace | Grants access to any resource in cluster |
| admin | Grants access to most resources in the namespace, including roles and role bindings, but does not grant access to the namespace itself nor does it grant access to  cluster roles or cluster role bindings | Grants access to most resources in cluster, including roles and role bindings, but does not grant access to namespaces nor does it grant access to cluster roles or cluster role bindings |
| edit | Grants access to most resources in the namespace but does not grant access to roles or role bindings | Grants access to most resources in the cluster but does not grant access to roles or role bindings |
| view | Grants access to read most resources in the namespace but does not grant write access nor does it grant read access to secrets | Grants access to read most resources in the cluster but does not grant write access nor does it grant read access to secrets |

Infra does not differentiate between `Role` and `ClusterRole`. Instead, Infra will create the appropriate binding based on whether the Grant is defined for a specific namespace or for the entire cluster.

<aside>
💡 Infra does not currently manage `Role` or `ClusterRole`. However, any existing Kubernetes `ClusterRole` can be referenced in an Infra grant. Kubernetes `Role` are not supported.

</aside>

### Example: Grant group `Everyone` the `view` role to a cluster

This Grant will provide the group `Everyone` read-only access into a cluster. Members of the group will be able to query Kubernetes resources but not modify any resources. Users will also not be able to query secrets as that require at least the `edit` role.

```bash
infra grants add --group Everyone --role view kubernetes.cluster
```

### Example: Grant group `Core` the `cluster-admin` role to a namespace

This Grant will provide the group `Core` super admin access into a namespace. Members of the group will be able to create, update, and delete any resource so long as the resources they’re modifying are exist in the namespace.

```bash
infra grants add --group Core --role cluster-admin kubernetes.cluster.namespace
```

## Revoking Access

<aside>
💡 Grants are built with an additive model. There is no plan currently to define grants which reduce access.

</aside>

To remove access to a resource, remove the grant providing access.

```bash
infra grants remove --group Core --role cluster-admin kubernetes.cluster.namespace
```
