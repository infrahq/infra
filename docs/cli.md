# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra list](#infra-list)
* [infra tokens create](#infra-tokens-create)
* [infra version](#infra-version)


## `infra login`

Login to Infra

```
infra login [HOST] [flags]
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

### Options inherited from parent commands

```
  -f, --config-file string   Infra configuration file path
  -H, --host string          Infra host
  -l, --log-level string     log level (default "info")
```

## `infra logout`

Logout Infra

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
  -f, --config-file string   Infra configuration file path
  -H, --host string          Infra host
  -l, --log-level string     log level (default "info")
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

### Options inherited from parent commands

```
  -f, --config-file string   Infra configuration file path
  -H, --host string          Infra host
  -l, --log-level string     log level (default "info")
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
  -f, --config-file string   Infra configuration file path
  -H, --host string          Infra host
  -l, --log-level string     log level (default "info")
```

## `infra version`

Display the Infra build version

```
infra version [flags]
```

### Options

```
      --client   Display client version only
  -h, --help     help for version
      --server   Display server version only
```

### Options inherited from parent commands

```
  -f, --config-file string   Infra configuration file path
  -H, --host string          Infra host
  -l, --log-level string     log level (default "info")
```

