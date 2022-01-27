# Configuration

For teams who require configuration to be stored in version control, Infra can be entirely configured as code.

## Usage

```
# Helm
helm install infra infrahq/infra --set-file config=config.yaml

# Linux, macOS, Windows
infra server -f config.yaml
```

## Reference

| Option                                               | Description                                                                                    |
|------------------------------------------------------|------------------------------------------------------------------------------------------------|
| `providers`                                          | Identity providers (see below)                                                                 |
| `groups`                                             | Groups and their grants (see below)                                                            |
| `users`                                              | Users and their grants (see below)                                                             |
| `secrets`                                            | Secrets configurations (see [Secrets](./secrets.md) for more info)                             |
| `keys`                                               | Encryption key configuration (see [Keys](./keys.md) for more info)                             |
| `dbEncryptionKey`                                    | Database encryption key                                                                                   |
| `dbEncryptionKeyProvider`                            | Database encryption key provider (default "native" – see [Keys](./keys.md) for more info)      |
| `dbFile`                                             | Path to database file (default "~/.infra/db")                                                  |
| `dbHost`                                             | Database host                                                                                  |
| `dbPort`                                             | Database host                                                                                  |
| `dbName`                                             | Database name                                                                                  |
| `dbUser`                                             | Database user                                                                                  |
| `dbParameters`                                       | Path to database file (default "~/.infra/db")                                                  |
| `dbPassword` ([secret](./secrets.md))                | Path to database file (default "~/.infra/db")                                                  |
| `enableCrashReporting`                               | Enable crash reporting (default true)                                                          |
| `enableTelemetry`                                    | Enable telemetry (default true)                                                                |
| `engineAPIToken` ([secret](./secrets.md))            | Engine API token secret (default "file:~/.infra/engine-api-token")                             |
| `rootAPIToken` ([secret](./secrets.md))              | Root API token secret (default "file:~/.infra/root-api-token")                                 |
| `sessionDuration`                                    | Session duration (default 12h0m0s)                                                             |
| `tlsCache`                                           | Directory to cache TLS certificates (default "~/.infra/tls")                                   |


### `providers`

List of identity providers used to synchronize users and groups.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `kind`         | Provider type (e.g. "okta")                  |
| `domain`       | provider domain                              |
| `clientID`     | OpenID client ID                             |
| `clientSecret` | OpenID client secret                         |
| `apiToken`     | Provider API token                           |

See [Identity Providers](./providers/) for a full list of configurable values.

### `groups`

List of groups to assign access.

| Parameter      | Description                                   |
|----------------|-----------------------------------------------|
| `name`         | Group name as stored in the identity provider |
| `grants`       | Role grants assigned to the user              |

### `users`

List of users to assign access.

| Parameter      | Description                                   |
|----------------|-----------------------------------------------|
| `email`        | User email as stored in the identity provider |
| `grants`       | Role grants assigned to the user              |

### `grants`

List of role grants to assign to an user or group.

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `name`         | Kubernetes role name                         |
| `kind`         | Kubernetes role kind (role, cluster-role)    |
| `destinations` | Destinations where this role binding applies |

### `secrets`

See [secrets](./secrets.md) for more information

## Full Example

```yaml
providers:
  - kind: okta
    domain: acme.okta.com
    clientID: 0oapn0qwiQPiMIyR35d6
    clientSecret: kubernetes:infra-okta/clientSecret

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
