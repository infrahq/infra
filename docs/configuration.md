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
  providers:
    [...]
  groups:
    [...]
  users:
    [...]
```

See [Helm Chart reference](./helm.md) for a complete list of options configurable through Helm.

Then, apply it to Infra:

```
helm -n infrahq upgrade -f values.yaml infra infrahq/infra
```

## Configurations

### Configuration File

Configuration values can be configured through configuration files. Support formats include `json`, `yaml`, `toml`, and `ini`. The configuration file must be called `config`, e.g. `config.json`, `config.yaml`, `config.ini`, etc.

Configuration files are searched in the following paths, in order:

* `.` (current working directory)
* `~/.infra` (user home directory)
* `/etc/infra`

Note: only the first configuration file found will be used.

Configuration file can also be explicitly supplied through environment variable `INFRA_CONFIG_FILE` or command line parameter `--config-file` or `-f`.

### Environment Variables

Most configuration values can be configured through environment variables. Environment variables start with `INFRA`, e.g. `INFRA_CONFIG_FILE`. Environment variables have higher precedence than values found in configuration files.

### Command Line Parameters

Most configuration values can be configured through command line parameters. Command line parameters have higher precedence than environment variables or configuration files.

See [CLI Reference](./cli.md) for a complete list of support command line parameters.

### Reference

| Configuration | Subcommand | Description                 | Default | Environment Variable | Command Line Parameter |
|---------------|------------|-----------------------------|---------|----------------------|------------------------|
| `host`        |            | Infra host URL              | `""`    | `INFRA_HOST`         | `--host`, `-H`         |
| `config-file` |            | Configuration file path     | `""`    | `INFRA_CONFIG_FILE`  | `--config-file`, `-f`  |
| `v`           |            | Log verbosity               | `0`     | `INFRA_V`            | `--v`, `-v`            |
| `timeout`     | `login`    | Login timeout               | `5m0s`  | `INFRA_TIMEOUT`      | `--timeout`, `-t`      |
| `client`      | `version`  | Display client version only | `false` | `INFRA_CLIENT`       | `--client`             |
| `server`      | `version`  | Display server version only | `false` | `INFRA_SERVER`       | `--server`             |


## Infra Configurations

First, create a config file `infra.yaml`:

```
providers:
  [...]
groups:
  [...]
users:
  [...]
```

Then, apply it to Infra:

```
helm -n infrahq upgrade --set-file=config=infra.yaml infra infrahq/infra
```

### Reference

#### `providers`

List of identity providers used to synchronize users and groups.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `kind`         | Provider type                                |
|                | Additional provider-specific parameters      |

See [Identity Providers](./providers/) for a full list of configurable values.

#### `groups`

List of groups to assign access.

| Parameter      | Description                                   |
|----------------|-----------------------------------------------|
| `name`         | Group name as stored in the identity provider |
| `grants`       | Grants assigned to the user                   |

#### `users`

List of users to assign access.

| Parameter      | Description                                   |
|----------------|-----------------------------------------------|
| `email`        | User email as stored in the identity provider |
| `grants`       | Grants assigned to the user                   |

#### `grants`

List of grants to assign to an user or group.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `name`         | Kubernetes role name                         |
| `kind`         | Kubernetes role kind (role, cluster-role)    |
| `destinations` | Destinations where this role binding applies |

#### `destinations`

List of infrastructure destination to synchronize access permissions.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `name`         | Destination name                             |
| `labels`       | Additional filter labels                     |
|                | Additional destination-specific parameters   |

See [Infrastructure Destinations](./destinations/) for a full list of configurable values.

### Full Example

```yaml
providers:
  - kind: okta
    domain: acme.okta.com
    clientID: 0oapn0qwiQPiMIyR35d6
    clientSecret: kubernetes:infra-okta/clientSecret
    apiToken: kubernetes:infra-okta/apiToken

groups:
  - name: administrators
    provider: okta
    grants:
      - name: cluster-admin
        kind: cluster-role
        destinations:
          - labels: [kubernetes]

users:
  - email: manager@example.com
    grants:
      - name: view
        kind: cluster-role
        destinations:
          - name: my-first-destination
            namespaces:
              - infrahq
              - development
          - labels: [kubernetes]

  - email: developer@example.com
    grants:
      - name: writer
        kind: role
        destinations:
          - name: my-first-destination
            namespaces:
              - development
```
