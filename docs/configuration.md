
# Configuration

## Overview

For teams who require configuration to be stored in version control, Infra can be managed via Helm values or a standalone file.

## Helm Values

Infra configuration can be added to Helm values under the `config` key.

First, create a `values.yaml`. If a `values.yaml` already exists, update it to include the following:

```
# values.yaml
---
config:
  sources: [...]
  groups: [...]
  users: [...]
```

See [Helm Chart reference](./helm.md) for a complete list of options configurable through Helm.

Then, apply it to Infra:

```
helm -n infrahq upgrade -f values.yaml infra infrahq/infra
```

## Standalone Configuration

First, create a config file `infra.yaml`:

```
sources: [...]
groups: [...]
users: [...]
```

Then, apply it to Infra:

```
helm -n infrahq upgrade --set-file=config=infra.yaml infra infrahq/infra
```

## Reference

### `sources`

List of identity sources used to synchronize users and groups.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `kind`         | Source type                                  |
|                | Additional source-specific parameters        |

See [Identity Sources](./sources/) for a full list of configurable values.

### `groups`

List of groups to assign access.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `name`         | Group name as stored in the identity source  |
| `roles`        | Roles assigned to the user                   |

### `users`

List of users to assign access.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `email`        | User email as stored in the identity source  |
| `roles`        | Roles assigned to the user                   |

### `roles`

List of roles to assign to an user or group.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `name`         | Role name                                    |
| `kind`         | Role type                                    |
| `destinations` | Destinations where this role binding applies |

### `destinations`

List of infrastructure destination to synchronize access permissions.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `name`         | Destination name                             |
|                | Additional destination-specific parameters   |

See [Infrastructure Destinations](./destinations/) for a full list of configurable values.

## Full Example

```yaml
sources:
  - kind: okta
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: kubernetes:infra-okta/clientSecret
    okta:
      api-token: kubernetes:infra-okta/apiToken

groups:
  - name: administrators
    source: okta
    roles:
      - name: cluster-admin
        kind: cluster-role
        destinations:
          - name: my-first-destination

users:
  - email: manager@example.com
    roles:
      - name: view
        kind: cluster-role
        destinations:
          - name: my-first-destination
            namespaces: 
              - infrahq
              - development
          - name: cluster-BBB
  - email: developer@example.com
    roles:
      - name: writer
        kind: role
        destinations:
          - name: my-first-destination
            namespaces:
              - development
```
