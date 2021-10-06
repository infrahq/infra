# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra list](#infra-list)
* [infra token](#infra-token)
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
  -h, --help   help for login
```

## `infra logout`

Logout of an Infra Registry

```
infra logout [flags]
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

## `infra token`

Generate a JWT token for connecting to a destination, e.g. Kubernetes

```
infra token DESTINATION [flags]
```

### Options

```
  -h, --help   help for token
```

## `infra version`

Display the Infra build version

```
infra version [flags]
```

### Options

```
  -h, --help   help for version
```

## `infra registry`

Start Infra Registry

```
infra registry [flags]
```

### Options

```
  -c, --config string                   config file
      --db string                       path to database file (default "~/.infra/infra.db")
  -h, --help                            help for registry
      --engine-api-key string   initial api key for adding destinations
      --root-api-key string             the root api key for privileged actions
      --sync-interval int               the interval (in seconds) at which Infra will poll sources for users and groups (default 30)
      --tls-cache string                path to directory to cache tls self-signed and Let's Encrypt certificates (default "~/.infra/cache")
      --ui                              enable ui
      --ui-proxy string                 proxy ui requests to this host
```

## `infra engine`

Start Infra Engine

```
infra engine [flags]
```

### Options

```
      --api-key string     api key
      --force-tls-verify   force TLS verification
  -h, --help               help for engine
  -n, --name string        cluster name
  -r, --registry string    registry hostname
      --tls-cache string   path to directory to cache tls self-signed and Let's Encrypt certificates (default "~/.infra/cache")
```

