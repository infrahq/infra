# CLI Reference

## Commands

* [infra users create](#infra-users-create)
* [infra users list](#infra-users-list)
* [infra users delete](#infra-users-delete)
* [infra providers list](#infra-providers-list)
* [infra providers create](#infra-providers-create)
* [infra providers delete](#infra-providers-delete)
* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra server](#infra-server)


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
infra users delete ID [flags]
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

## `infra providers create`

Create a provider connection

```
infra providers create KIND [flags]
```

### Examples

```
$ infra providers create okta \
	--domain example.okta.com \
	--apiToken 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd \
	--clientID 0oapn0qwiQPiMIyR35d6 \
	--clientSecret jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
```

### Options

```
      --api-token string       Api Token
      --client-id string       Client ID for single sign on
      --client-secret string   Client Secret for single sign on
      --domain string          Identity provider domain (e.g. example.okta.com)
  -h, --help                   help for create
```

### Options inherited from parent commands

```
  -i, --insecure   skip TLS verification
```

## `infra providers delete`

Delete a provider connection

```
infra providers delete ID [flags]
```

### Examples

```
$ infra providers delete n7bha2pxjpa01a
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
  -i, --insecure   skip TLS verification
```

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

## `infra logout`

Log out of Infra server

```
infra logout [flags]
```

### Options

```
  -h, --help   help for logout
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

