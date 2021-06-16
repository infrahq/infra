# Granting & revoking access

## Roles

Roles are high-level sets of permissions that grant access to specific clusters.

### Default Roles

| Role                    | Description                        |
| :---------------------- | :------------------------------    |
| kubernetes.viewer       | Read-only for most resources       |
| kubernetes.editor       | Read & write most resources        |
| kubernetes.admin        | Read & write any resource          |

## Granting access via a role

```
infra grant <user> <cluster> --role <role>
```

For example, to provide a user read-only access to a cluster:

```
infra grant user@example.com example-cluster --role kubernetes.viewer
```

## Revoke access via a role

To revoke all roles:

```
infra revoke user@example.com example-cluster
```

To revoke a specific role:

```
infra revoke user@example.com example-cluster --role kubernetes.viewer
```

## Granting roles for Infra itself

Infra includes it's own set of roles for itself. By default, any usser added has the `infra.member` role.

| Role                    | Description                        |
| :---------------------- | :------------------------------    |
| infra.member            | List & login to clusters via Infra |
| infra.owner             | Full access to Infra               |

### Grant a user the `infra.owner` role

```
infra grant user@example.com infra --role infra.owner
```

### Revoke the `infra.owner` role

```
infra revoke user@example.com --role infra.owner
```