# CLI Reference

## Commands

* [infra list](#infra-list)
* [infra status](#infra-status)
* [infra users](#infra-users)
* [infra groups](#infra-groups)
* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra creds](#infra-creds)
* [infra registry](#infra-registry)
* [infra engine](#infra-engine)
* [infra version](#infra-version)


## `infra list`

List destinations

```
infra list [flags]
```

### Options

```
  -h, --help    help for list
```

## `infra status`

Show the status of all connected destinations

```
infra status [flags]
```

### Options

```
  -h, --help    help for status
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

## `infra groups`

List groups

```
infra groups [flags]
```

### Options

```
  -h, --help   help for groups
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

## `infra creds`

Get a token for a specific destination. Typically only used internally.

```
infra creds DESTINATION
```

### Examples

```
$ infra creds kubernetes
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
      --sync-interval int       the interval (in seconds) at which Infra will poll sources for users and groups (default 30)
      --tls-cache string        path to directory to cache tls self-signed and Let's Encrypt certificates (default "$HOME/.infra/cache")
      --ui                      enable ui
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

