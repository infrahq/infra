# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra users create](#infra-users-create)
* [infra users list](#infra-users-list)
* [infra users delete](#infra-users-delete)
* [infra providers list](#infra-providers-list)
* [infra server](#infra-server)


## `infra login`

Log in to Infra server

```
infra login HOST [flags]
```

### Examples

```
$ infra login infra.example.com
```

### Options

```
  -h, --help       help for login
  -i, --insecure   skip TLS verification
```

## `infra users create`

create a user

```
infra users create EMAIL PASSWORD [flags]
```

### Examples

```
$ infra users create admin@example.com p4assw0rd
```

### Options

```
  -h, --help   help for create
```

### Options inherited from parent commands

```
  -i, --insecure   skip TLS verification
```

## `infra users list`

List users

```
infra users list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -i, --insecure   skip TLS verification
```

## `infra users delete`

delete a user

```
infra users delete EMAIL [flags]
```

### Examples

```
$ infra users delete user@example.com
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
  -i, --insecure   skip TLS verification
```

## `infra providers list`

List providers

```
infra providers list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -i, --insecure   skip TLS verification
```

## `infra server`

Start Infra server

```
infra server [flags]
```

### Options

```
  -c, --config string      server config file
      --db string          path to database file (default "/Users/jmorgan/.infra/infra.db")
  -h, --help               help for server
      --tls-cache string   path to directory to cache tls self-signed and Let's Encrypt certificates (default "/Users/jmorgan/.infra/cache")
      --ui                 enable experimental UI
      --ui-proxy           proxy ui requests to localhost:3000
```

