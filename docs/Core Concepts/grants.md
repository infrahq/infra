# Grants

Infra grants are based on the simple relationships between subject (users, groups, or machines), privilege (roles, permissions), and resource (Kubernetes clusters, etc.). How grants are interpreted by the resource is an implementation detail of the specific connector.

For example, Kubernetes has an RBAC using ClusterRoles and Roles. Therefore, it is the job of the Infra Kubernetes Connector to translate grants to Kubernetes ClusterRoles and ClusterRoleBindings, and Roles and RoleBindings.

Other integrations will have different authorization primitives and will therefore require different interpretations of grants. Some integrations may not implement authorization at all or have a significantly different implementation so the connector will need to implement its own authorization using the grant primitives.

Grants are implemented in an additive model where the base configuration is to not provide any access. As grants are applied to Infra, subjects will progressively gain access to Infra and connected destinations.

## Infra Grants

Infra grants are a special type of Grant where the resource is Infra itself. This model is used to give subjects access to the Infra API. The privilege for Infra grants currently comprise of three roles: `admin`, `user`, and `connector`.

Each authenticated API call will check the caller has a role required to access the requested resource. If the caller has the necessary role to access the resource, the request is fulfilled and the result is returned. If the caller does *not* have the necessary role to access the resource, the request is rejected and an error is returned.

The `connector` role is special in that it is intended to be used solely by a connector and provides the necessary resource for the connector to configure itself. Users should *not* create grants using this role.

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

For details on Kubernetes grants, see [Connect Destinations: Kubernetes](../Connectors/connect-destinations/connect-destinations-kubernetes.md#grants).

## Revoking Access

> :exclamation: Grants are built with an additive model. There is no plan currently to define grants which reduce access.

To remove access to a resource, remove the grant providing access.

```bash
infra grants remove --group Core --role cluster-admin kubernetes.cluster.namespace
```
