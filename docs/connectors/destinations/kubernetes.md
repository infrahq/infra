# Kubernetes

## Installing the connector

### 1. Install the connector via `helm`:

```
infra destinations add kubernetes.example
```

Run the output `helm` command on the Kubernetes cluster you want to connect to Infra.

## Granting access

Once you've connected a cluster, you can grant access via `infra grants add`:

```
# grant access to a user
infra grants add fisher@example.com kubernetes.example --role admin

# grant access to a group
infra grants add engineering kubernetes.example --role view
```

### Supported roles

| Role | Access level |
| --- | --- |
| cluster-admin | Grants access to any resource |
| admin | Grants access to most resources in the namespace, including roles and role bindings, but does not grant access to the namespace itself nor does it grant access to cluster roles or cluster role bindings |
| edit | Grants access to most resources in the namespace but does not grant access to roles or role bindings
| view | Grants access to read most resources in the namespace but does not grant write access nor does it grant read access to secrets |

### Example: Grant user `dev@example.com` the `view` role to a cluster

This command will grant the user `dev@example.com` read-only access into a cluster, giving that user the privileges to query Kubernetes resources but not modify any resources.

```bash
infra grants add --user dev@example.com --role view kubernetes.cluster
```

### Example: Grant user `ops@example.com` the `admin` role to a namespace

This command will grant the user `ops@example.com` admin access into a namespace, giving that user the privileges to create, update, and delete any resource so long as the resources they’re modifying exist in the namespace.

```bash
infra grants add --user ops@example.com --role admin kubernetes.cluster.namespace
```

### Example: Revoke from the user `ops@example.com` the `admin` role to a namespace

This command will remove the `admin` role, granted in the previous example, from `ops@example.com`.

```bash
infra grants remove --user ops@example.com --role cluster-admin kubernetes.cluster.namespace
```

### Roles

| Role | Access level |
| --- | --- |
| cluster-admin | Grants access to any resource |
| admin | Grants access to most resources in the namespace, including roles and role bindings, but does not grant access to the namespace itself nor does it grant access to cluster roles or cluster role bindings |
| edit | Grants access to most resources in the namespace but does not grant access to roles or role bindings
| view | Grants access to read most resources in the namespace but does not grant write access nor does it grant read access to secrets |