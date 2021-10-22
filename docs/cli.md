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
  -c, --client   Display client version only
  -h, --help     help for version
  -r, --infra    Display infra version only
```

