# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra user create](#infra-user-create)
* [infra user list](#infra-user-list)
* [infra user delete](#infra-user-delete)
* [infra source list](#infra-source-list)
* [infra source create](#infra-source-create)
* [infra source delete](#infra-source-delete)
* [infra destination list](#infra-destination-list)
* [infra destination create](#infra-destination-create)
* [infra server](#infra-server)
* [infra engine](#infra-engine)


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
  -h, --help              help for login
  -k, --skip-tls-verify   skip TLS verification
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

## `infra user create`

create a user

```
infra user create EMAIL PASSWORD [flags]
```

### Examples

```
$ infra user create admin@example.com password
```

### Options

```
  -h, --help   help for create
```

## `infra user list`

List users

```
infra user list [flags]
```

### Options

```
  -h, --help   help for list
```

## `infra user delete`

delete a user

```
infra user delete USER [flags]
```

### Examples

```
$ infra user delete user@example.com
```

### Options

```
  -h, --help   help for delete
```

## `infra source list`

List sources

```
infra source list [flags]
```

### Options

```
  -h, --help   help for list
```

## `infra source create`

Connect an identity source

```
infra source create KIND [flags]
```

### Examples

```
$ infra source create okta \
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
      --domain string          Domain (e.g. example.okta.com)
  -h, --help                   help for create
```

## `infra source delete`

Delete an identity source

```
infra source delete ID [flags]
```

### Examples

```
$ infra source delete n7bha2pxjpa01a
```

### Options

```
  -h, --help   help for delete
```

## `infra destination list`

List clusters

```
infra destination list [flags]
```

### Options

```
  -h, --help   help for list
```

## `infra destination create`

Connect a destination

```
infra destination create NAME [flags]
```

### Options

```
  -h, --help   help for create
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
      --api-key string    api key
  -h, --help              help for engine
  -n, --name string       cluster name
  -r, --registry string   registry hostname
  -k, --skip-tls-verify   skip TLS verification
```

