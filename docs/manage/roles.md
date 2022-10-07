---
title: Working with Roles
position: 3
---

# Working with Roles

Grant roles to users in Infra to give that user a certain level of access to a cluster or namespace. [Learn more about Granting Roles to Users](grants.md).

Roles supported by a connector are defined only in the context of the connected cluster. Infra supports the following roles by default:

| Role            | Access level                                                                                                                                                            |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cluster-admin` | Grants access to any resource                                                                                                                                           |
| `admin`         | Grants access to most resources, including roles and role bindings, but does not grant access to cluster-level resources such as cluster roles or cluster role bindings |
| `edit`          | Grants access to most resources in the namespace but does not grant access to roles or role bindings                                                                    |
| `view`          | Grants access to read most resources in the namespace but does not grant write access nor does it grant read access to secrets                                          |
| `logs`          | Grants access to pod logs                                                                                                                                               |
| `exec`          | Grants access to `kubectl exec`                                                                                                                                         |
| `port-forward`  | Grants access to `kubectl port-forward`                                                                                                                                 |

## Custom Kubernetes Roles

If the provided roles are not sufficient, additional roles can be configured to integrate with Infra. To add a new role, create a ClusterRole in a connected cluster with label `app.infrahq.com/include-role=true`.

```bash
kubectl create clusterrole example --verb=get --resource=pods
kubectl label clusterrole/example app.infrahq.com/include-role=true
```

## Additional Information

- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
