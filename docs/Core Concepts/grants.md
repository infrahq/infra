# Grants

Infra Grants are based on the simple relationships between Subject (users, groups), Privilege (roles, permissions), and Resource. How grants are interpreted by the resource is an implementation detail of the specific connector. 

For example, Kubernetes has an RBAC using ClusterRoles and Roles. Therefore, it is the job of the Infra Kubernetes Connector to translate Grants to Kubernetes ClusterRoles and ClusterRoleBindings, and Roles and RoleBindings. 

Other integrations will have different authorization primitives and will therefore require different interpretations of Grants. Some integrations may not implement authorization at all or have a significantly different implementation so the connector will need to implement its own authorization using the grant primitives.

Grants are implemented in an additive model where the base configuration is to not provide any access. As Grants are applied to Infra, subjects will progressively gain access to Infra and connected destinations.

## Infra Grants

Infra Grants are a special type of Grant where the resource is Infra itself. This model is used to give subjects access to the Infra API. The Privilege for Infra Grants currently comprise of three roles: `admin`, `user`, and `connector`. 

Each authenticated API call will check the caller has a role required to access the requested resource. If the caller has the necessary role to access the resource, the request is fulfilled and the result is returned. If the caller does *not* have the necessary role to access the resource, the request is rejected and an error is returned.

The `connector` role is special in that it is intended to be used solely by a connector and provides the necessary resource for the connector to configure itself. Users should *not* create Grants using this role.

### Example: Grant user `admin@example.com` the admin role to Infra

This Grant will provide `admin@example.com` full access to the Infra API, including creating additional grants, managing identity providers, managing destinations, and managing other users.

```bash
infra grants add --user admin@example.com --role admin infra
```

### Example: Grant user `dev@example.com` the user role to Infra

This Grant will provide `dev@example.com` *some* access to the Infra API, including logging in and using a destination they have been granted access to, listing destinations, and updating their own user. It does *not* include access to creating additional grants, managing identity providers, managing destinations, or managing other users.

```bash
infra grants add --user dev@example.com --role user infra
```

## Kubernetes Grants

As mentioned in the introduction, Kubernetes Grants are built on top of Kubernetes RBAC which consists of `Role` and `ClusterRole` and `RoleBinding` and `ClusterRoleBinding`. For more detailed explanation of these concepts, checkout the official documentation:

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
