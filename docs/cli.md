# CLI Reference

## Commands

* [infra list](#infra-list)
* [infra users](#infra-users)
* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra registry](#infra-registry)
* [infra engine](#infra-engine)
* [infra version](#infra-version)


## `infra list`

List clusters

```
infra list [flags]
```

### Options

```
  -h, --help   help for list
```

## `infra users`

List users

```
infra users [flags]
```

### Options

```
  -h, --help   help for users
```

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

## `infra registry`

Start Infra Registry

```
infra registry [flags]
```

### Options

```
  -c, --config string           config file
      --db string               path to database file (default "$HOME/.infra/infra.db")
  -h, --help                    help for registry
      --initial-apikey string   initial api key for adding destinations
      --tls-cache string        path to directory to cache tls self-signed and Let's Encrypt certificates (default "$HOME/.infra/cache")
      --ui-proxy string         proxy ui requests to this host
```

## `infra engine`

Start Infra Engine

```
infra engine [flags]
```

### Options

```
      --api-key string     api key
  -e, --endpoint string    cluster endpoint
      --force-tls-verify   force TLS verification
  -h, --help               help for engine
  -n, --name string        cluster name
  -r, --registry string    registry hostname
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

