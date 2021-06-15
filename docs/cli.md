# CLI Reference

## Commands

* [infra list](#infra-list)
* [infra grant](#infra-grant)
* [infra revoke](#infra-revoke)
* [infra inspect](#infra-inspect)
* [infra users create](#infra-users-create)
* [infra users list](#infra-users-list)
* [infra users delete](#infra-users-delete)
* [infra providers list](#infra-providers-list)
* [infra providers create](#infra-providers-create)
* [infra providers delete](#infra-providers-delete)
* [infra signup](#infra-signup)
* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra server](#infra-server)
* [infra engine](#infra-engine)


## `infra list`

List clusters

```
infra list [flags]
```

### Options

```
  -h, --help   help for list
```

## `infra grant`

Grant access to a resource

```
infra grant USER RESOURCE [flags]
```

### Examples

```
$ infra grant user@example.com production --role kubernetes.editor
```

### Options

```
  -h, --help          help for grant
  -r, --role string   role
```

## `infra revoke`

Revoke access to a resource

```
infra revoke USER RESOURCE [flags]
```

### Examples

```
$ infra revoke user@example.com production
$ infra revoke user@example.com production --role kubernetes.editor
```

### Options

```
  -h, --help          help for revoke
  -r, --role string   role
```

## `infra inspect`

Inspect access for a resource or user

```
infra inspect CLUSTER|USER [flags]
```

### Options

```
  -h, --help   help for inspect
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

## `infra users list`

List users

```
infra users list [flags]
```

### Options

```
  -h, --help   help for list
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

## `infra providers list`

List providers

```
infra providers list [flags]
```

### Options

```
  -h, --help   help for list
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

## `infra signup`

Create the admin user for a new Infra Server

```
infra signup HOST [flags]
```

### Examples

```
$ infra signup infra.example.com
```

### Options

```
  -h, --help       help for signup
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

## `infra engine`

Start Infra engine

```
infra engine [flags]
```

### Options

```
      --api-key string   api key
  -h, --help             help for engine
  -i, --insecure         skip TLS verification
  -n, --name string      cluster name
  -s, --server string    server hostname
```

