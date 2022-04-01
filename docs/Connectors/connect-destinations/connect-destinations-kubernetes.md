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

3. Repeat the previous step for each new Kubernetes cluster being added to Infra.

## Upgrade with Helm

See [Upgrading Infra Kubernetes Connector](../../Operator%20Guides/upgrading-infra.md).

## Grants

Kubernetes grants are built on top of Kubernetes RBAC which consists of `Role`, `ClusterRole`, `RoleBinding` and `ClusterRoleBinding`. For more detailed explanation of these concepts, checkout the official documentation:

[Using RBAC Authorization](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)

To summarize:

- `Role` is a set of permissions applied to a specific namespace. It can only be referenced in the namespace it is created in.
- `ClusterRole` is a set of permissions applied to the entire cluster. It can also be applied to a namespaced resourced within a specific namespace.
- `RoleBinding` binds a set of subjects (users, groups, system accounts) to a `Role` in a given namespace.
- `ClusterRoleBinding` binds a set of subjects (users, groups, system accounts) to a `ClusterRole` in a given namespace or in the entire cluster.

There are several default user-facing `Role` and `ClusterRole` objects that are set up in your Kubernetes cluster. These objects should be sufficient unless you want to create more complex RBAC configurations.

| Name | With Namespace | Without Namespace |
| --- | --- | --- |
| cluster-admin | Grants access to any resource in the namespace | Grants access to any resource in the cluster |
| admin | Grants access to most resources in the namespace, including roles and role bindings, but does not grant access to the namespace itself nor does it grant access to  cluster roles or cluster role bindings | Grants access to most resources in the cluster, including roles and role bindings, but does not grant access to namespaces nor does it grant access to cluster roles or cluster role bindings |
| edit | Grants access to most resources in the namespace but does not grant access to roles or role bindings | Grants access to most resources in the cluster but does not grant access to roles or role bindings |
| view | Grants access to read most resources in the namespace but does not grant write access nor does it grant read access to secrets | Grants access to read most resources in the cluster but does not grant write access nor does it grant read access to secrets |

Infra will translate Kubernetes grants to a `ClusterRoleBinding`. If a grant is defined for a specific namespace, the `ClusterRoleBinding` will be defined for that namespace. Otherwise, it is applied to the entire cluster.

> :exclamation: Infra does not currently manage `Role` or `ClusterRole` objects. However, existing Kubernetes `ClusterRole` objects can be referenced in an Infra grant. Kubernetes `Role` objects are not supported.

### Example: Grant user `dev@example.com` the `view` role to a cluster

This command will grant the user `dev@example.com` read-only access into a cluster, giving that user the privileges to query Kubernetes resources but not modify any resources. The user will also not be able to query secrets as that require at least the `edit` role.

```bash
infra grants add --user dev@example.com --role view kubernetes.cluster
```

### Example: Grant user `ops@example.com` the `cluster-admin` role to a namespace

This command will grant the user `ops@example.com` super admin access into a namespace, giving that user the privileges to create, update, and delete any resource so long as the resources they’re modifying exist in the namespace.

```bash
infra grants add --user ops@example.com --role cluster-admin kubernetes.cluster.namespace
```

### Example: Revoke from the user `ops@example.com` the `cluster-admin` role to a namespace

This command will remove the `cluster-admin` role, granted in the previous example, from `ops@example.com`.

```bash
infra grants remove --user ops@example.com --role cluster-admin kubernetes.cluster.namespace
```
