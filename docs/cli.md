# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra user create](#infra-user-create)
* [infra user list](#infra-user-list)
* [infra user delete](#infra-user-delete)
* [infra destination list](#infra-destination-list)
* [infra source list](#infra-source-list)
* [infra source create](#infra-source-create)
* [infra source delete](#infra-source-delete)
* [infra apikey list](#infra-apikey-list)
* [infra registry](#infra-registry)
* [infra engine](#infra-engine)


## `infra login`

Log in to an Infra Registry

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

Log out of an Infra Registry

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

## `infra destination list`

List destinations

```
infra destination list [flags]
```

### Options

```
  -h, --help   help for list
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
	--api-token 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd \
	--client-id 0oapn0qwiQPiMIyR35d6 \
	--client-secret jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
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

## `infra apikey list`

List API Keys

```
infra apikey list [flags]
```

### Options

```
  -h, --help   help for list
```

## `infra registry`

Start Infra Registry

```
infra registry [flags]
```

### Options

```
  -c, --config string           config file
      --db string               path to database file (default "/Users/jmorgan/.infra/infra.db")
  -h, --help                    help for registry
      --initial-apikey string   initial api key for adding destinations
      --tls-cache string        path to directory to cache tls self-signed and Let's Encrypt certificates (default "/Users/jmorgan/.infra/cache")
```

## `infra engine`

Start Infra Engine

```
infra engine [flags]
```

### Options

```
      --api-key string    api key
  -h, --help              help for engine
  -n, --name string       cluster name
  -r, --registry string   registry hostname
  -k, --skip-tls-verify   skip TLS verification (default true)
```

