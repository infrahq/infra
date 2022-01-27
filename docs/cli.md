# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra use](#infra-use)
* [infra list](#infra-list)
* [infra tokens create](#infra-tokens-create)
* [infra server](#infra-server)
* [infra engine](#infra-engine)
* [infra version](#infra-version)


## `infra login`

Login to Infra

```
infra login [HOST] [flags]
```

### Examples

```
$ infra login
```

### Options

```
  -h, --help   help for login
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra logout`

Logout of Infra

```
infra logout [flags]
```

### Examples

```
$ infra logout
```

### Options

```
  -h, --help   help for logout
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra use`

Connect to infrastructure

```
infra use [INFRASTRUCTURE] [flags]
```

### Options

```
  -h, --help               help for use
  -l, --labels strings     Labels
  -n, --namespace string   Namespace
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra list`

List infrastructure

```
infra list [flags]
```

### Options

```
  -a, --all    list all infrastructure (default shows infrastructure you have access to)
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra tokens create`

Create a JWT token for connecting to a destination, e.g. Kubernetes

```
infra tokens create DESTINATION [flags]
```

### Options

```
  -h, --help   help for create
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra server`

Start Infra Server

```
infra server [flags]
```

### Options

```
  -f, --config-file string                  Server configuration file
      --db-encryption-key string            Database encryption key (default "/Users/jmorgan/.infra/key")
      --db-encryption-key-provider string   Database encryption key provider (default "native")
      --db-file string                      Path to database file (default "/Users/jmorgan/.infra/db")
      --db-host string                      Database host
      --db-name string                      Database name
      --db-parameters string                Database additional connection parameters
      --db-password string                  Database password (secret)
      --db-port int                         Database port (default 5432)
      --db-user string                      Database user
      --enable-crash-reporting              Enable crash reporting (default true)
      --enable-telemetry                    Enable telemetry (default true)
      --engine-api-token string             Engine API token (secret) (default "file:/Users/jmorgan/.infra/engine-api-token")
  -h, --help                                help for server
      --providers-sync-interval duration    Interval at which Infra will poll identity providers for users and groups (default 1m0s)
      --root-api-token string               Root API token (secret) (default "file:/Users/jmorgan/.infra/root-api-token")
  -d, --session-duration duration           Session duration (default 12h0m0s)
      --tls-cache string                    Directory to cache TLS certificates (default "/Users/jmorgan/.infra/tls")
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra engine`

Start Infra Engine

```
infra engine [flags]
```

### Options

```
      --api-token string     Engine API token (use file:// to load from a file)
  -f, --config-file string   Engine config file
  -h, --help                 help for engine
  -k, --kind string          Destination kind (default "kubernetes")
  -n, --name string          Destination name
      --server string        Infra Server hostname
      --skip-tls-verify      Skip TLS verification (default true)
      --tls-cache string     Path to cache self-signed and Let's Encrypt TLS certificates
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra version`

Display the Infra version

```
infra version [flags]
```

### Options

```
  -h, --help   help for version
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

