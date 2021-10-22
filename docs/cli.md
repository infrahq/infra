# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra list](#infra-list)
* [infra tokens create](#infra-tokens-create)
* [infra version](#infra-version)
* [infra registry](#infra-registry)
* [infra engine](#infra-engine)


## `infra login`

Login to an Infra Registry

```
infra login REGISTRY [flags]
```

### Examples

```
$ infra login infra.example.com
```

### Options

```
  -h, --help               help for login
  -t, --timeout duration   login timeout (default 5m0s)
```

## `infra logout`

Logout of an Infra Registry

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

## `infra list`

List destinations

```
infra list [flags]
```

### Options

```
  -h, --help   help for list
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

## `infra version`

Display the Infra build version

```
infra version [flags]
```

### Options

```
  -c, --client     Display client version only
  -h, --help       help for version
  -r, --registry   Display registry version only
```

## `infra registry`

Start Infra Registry

```
infra registry [flags]
```

### Options

```
  -c, --config string            config file
      --db string                path to database file (default "~/.infra/infra.db")
      --enable-crash-reporting   enable crash reporting (default true)
      --enable-telemetry         enable telemetry (default true)
      --engine-api-key string    engine registration API key
  -h, --help                     help for registry
      --root-api-key string      root API key
      --sync-interval int        the interval (in seconds) at which Infra will poll sources for users and groups (default 30)
      --tls-cache string         path to directory to cache tls self-signed and Let's Encrypt certificates (default "~/.infra/cache")
      --ui                       enable ui
      --ui-proxy string          proxy ui requests to this host
```

## `infra engine`

Start Infra Engine

```
infra engine [flags]
```

### Options

```
      --engine-api-key string   engine registration API key
      --force-tls-verify        force TLS verification
  -h, --help                    help for engine
  -n, --name string             cluster name
  -r, --registry string         registry hostname
      --tls-cache string        path to directory to cache tls self-signed and Let's Encrypt certificates (default "~/.infra/cache")
```

