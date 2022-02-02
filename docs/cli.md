# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra list](#infra-list)
* [infra use](#infra-use)
* [infra access list](#infra-access-list)
* [infra access grant](#infra-access-grant)
* [infra access revoke](#infra-access-revoke)
* [infra destinations list](#infra-destinations-list)
* [infra destinations add](#infra-destinations-add)
* [infra destinations remove](#infra-destinations-remove)
* [infra providers list](#infra-providers-list)
* [infra providers add](#infra-providers-add)
* [infra providers remove](#infra-providers-remove)
* [infra tokens create](#infra-tokens-create)
* [infra import](#infra-import)
* [infra info](#infra-info)
* [infra server](#infra-server)
* [infra engine](#infra-engine)
* [infra version](#infra-version)


## `infra login`

Login to Infra

```
infra login [HOST] [flags]
```

### Examples

```
$ infra login
```

### Options

```
  -h, --help   help for login
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra logout`

Logout of Infra

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
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra list`

List destinations and your access

```
infra list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra use`

Connect to a destination

```
infra use [DESTINATION] [flags]
```

### Examples

```

# Connect to a Kubernetes cluster
infra use kubernetes.development

# Connect to a Kubernetes namespace
infra use kubernetes.development.kube-system
		
```

### Options

```
  -h, --help   help for use
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra access list`

List access

```
infra access list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra access grant`

Grant access

```
infra access grant DESTINATION [flags]
```

### Examples

```

# Grant user admin access to a cluster
infra grant -u suzie@acme.com -r admin kubernetes.production 

# Grant group admin access to a namespace
infra grant -g Engineering -r admin kubernetes.production.default

# Grant user admin access to infra itself
infra grant -u admin@acme.com -r admin infra

```

### Options

```
  -g, --group string      Group to grant access to
  -h, --help              help for grant
  -p, --provider string   Provider from which to grant user access to
  -r, --role string       Role to grant
  -u, --user string       User to grant access to
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra access revoke`

Revoke access

```
infra access revoke DESTINATION [flags]
```

### Options

```
  -g, --group string      Group to revoke access from
  -h, --help              help for revoke
  -p, --provider string   Provider from which to revoke access from
  -r, --role string       Role to revoke
  -u, --user string       User to revoke access from
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra destinations list`

List connected destinations

```
infra destinations list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra destinations add`

Connect a destination

```
infra destinations add TYPE NAME [flags]
```

### Options

```
  -h, --help   help for add
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra destinations remove`

Disconnect a destination

```
infra destinations remove DESTINATION [flags]
```

### Options

```
  -h, --help   help for remove
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra providers list`

List identity providers

```
infra providers list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra providers add`

Connect an identity provider

```
infra providers add NAME [flags]
```

### Options

```
      --client-id string       OpenID Client ID
      --client-secret string   OpenID Client Secret
  -h, --help                   help for add
      --url string             url or domain (e.g. acme.okta.com)
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra providers remove`

Disconnect an identity provider

```
infra providers remove PROVIDER [flags]
```

### Options

```
  -h, --help   help for remove
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra tokens create`

Create a token

```
infra tokens create [flags]
```

### Options

```
  -h, --help   help for create
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra import`

Import an infra server configuration

```
infra import [FILE] [flags]
```

### Options

```
  -h, --help      help for import
      --replace   replace any existing configuration
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra info`

Display the info about the current session

```
infra info [flags]
```

### Options

```
  -h, --help   help for info
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra server`

Start Infra Server

```
infra server [flags]
```

### Options

```
  -f, --config-file string                  Server configuration file
      --db-encryption-key string            Database encryption key (default "/Users/jmorgan/.infra/key")
      --db-encryption-key-provider string   Database encryption key provider (default "native")
      --db-file string                      Path to database file (default "/Users/jmorgan/.infra/db")
      --db-host string                      Database host
      --db-name string                      Database name
      --db-parameters string                Database additional connection parameters
      --db-password string                  Database password (secret)
      --db-port int                         Database port (default 5432)
      --db-user string                      Database user
      --enable-crash-reporting              Enable crash reporting (default true)
      --enable-telemetry                    Enable telemetry (default true)
      --access-key string                   Access key (secret) (default "file:/Users/jmorgan/.infra/access-key")
  -h, --help                                Help for server
      --admin-access-key string             Admin access key (secret) (default "file:/Users/jmorgan/.infra/admin-access-key")
  -d, --session-duration duration           Session duration (default 12h0m0s)
      --tls-cache string                    Directory to cache TLS certificates (default "/Users/jmorgan/.infra/tls")
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra engine`

Start Infra Engine

```
infra engine [flags]
```

### Options

```
      --access-key string     Infra access key (use file:// to load from a file)
  -f, --config-file string   Engine config file
  -h, --help                 help for engine
  -n, --name string          Destination name
      --server string        Infra Server hostname
      --skip-tls-verify      Skip TLS verification (default true)
      --tls-cache string     Path to cache self-signed and Let's Encrypt TLS certificates
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

## `infra version`

Display the Infra version

```
infra version [flags]
```

### Options

```
  -h, --help   help for version
```

### Options inherited from parent commands

```
      --log-level string   Log level (error, warn, info, debug) (default "info")
```

