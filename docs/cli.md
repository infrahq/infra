# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra list](#infra-list)
* [infra use](#infra-use)
* [infra access list](#infra-access-list)
* [infra access grant](#infra-access-grant)
* [infra access revoke](#infra-access-revoke)
* [infra keys list](#infra-keys-list)
* [infra keys create](#infra-keys-create)
* [infra keys delete](#infra-keys-delete)
* [infra destinations list](#infra-destinations-list)
* [infra destinations add](#infra-destinations-add)
* [infra destinations remove](#infra-destinations-remove)
* [infra providers list](#infra-providers-list)
* [infra providers add](#infra-providers-add)
* [infra providers remove](#infra-providers-remove)
* [infra machines create](#infra-machines-create)
* [infra machines list](#infra-machines-list)
* [infra machines remove](#infra-machines-remove)
* [infra tokens create](#infra-tokens-create)
* [infra import](#infra-import)
* [infra info](#infra-info)
* [infra server](#infra-server)
* [infra engine](#infra-engine)
* [infra version](#infra-version)


## `infra login`

Login to Infra server

```
infra login [SERVER] [flags]
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
  -f, --force   logout and remove context
  -h, --help    help for logout
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra use`

Connect to a destination

```
infra use DESTINATION [flags]
```

### Examples

```

# Connect to a Kubernetes cluster
$ infra use kubernetes.development

# Connect to a Kubernetes namespace
$ infra use kubernetes.development.kube-system
		
```

### Options

```
  -h, --help   help for use
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra access list`

List access

```
infra access list [DESTINATION] [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra access grant`

Grant access

```
infra access grant DESTINATION [flags]
```

### Examples

```

# Grant user admin access to a cluster
$ infra access grant -u suzie@acme.com -r admin kubernetes.production

# Grant group admin access to a namespace
$ infra access grant -g Engineering -r admin kubernetes.production.default

# Grant user admin access to infra itself
$ infra access grant -u admin@acme.com -r admin infra

```

### Options

```
  -g, --group string      Group to grant access to
  -h, --help              help for grant
  -m, --machine string    Machine to grant access to
  -p, --provider string   Provider from which to grant user access to
  -r, --role string       Role to grant
  -u, --user string       User to grant access to
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
  -m, --machine string    Machine to revoke access from
  -p, --provider string   Provider from which to revoke access from
  -r, --role string       Role to revoke
  -u, --user string       User to revoke access from
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra keys list`

List access keys

```
infra keys list [flags]
```

### Options

```
  -h, --help             help for list
  -m, --machine string   The name of a machine to list access keys for
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra keys create`

Create an access key for authentication

```
infra keys create ACCESS_KEY_NAME MACHINE_NAME [flags]
```

### Examples

```

# Create an access key for the machine "wall-e" called main that expires in 12 hours and must be used every hour to remain valid
infra keys create main wall-e 12h --extension-deadline=1h

```

### Options

```
  -e, --extension-deadline string   A specified deadline that an access key must be used within to remain valid
  -h, --help                        help for create
  -t, --ttl string                  The total time that an access key will be valid for
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra keys delete`

Delete access keys

```
infra keys delete ACCESS_KEY_NAME [flags]
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra providers add`

Add an identity provider

### Synopsis


Add an identity provider for users to authenticate.

NAME: The name of the identity provider (e.g. okta)
URL: The base URL of the domain used to login with the identity provider (e.g. acme.okta.com)
CLIENT_ID: The Infra application OpenID Connect client ID
CLIENT_SECRET: The Infra application OpenID Connect client secret
		

```
infra providers add NAME URL CLIENT_ID CLIENT_SECRET [flags]
```

### Options

```
  -h, --help   help for add
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra machines create`

Create a machine identity, e.g. a service that needs to access infrastructure

```
infra machines create NAME [flags]
```

### Options

```
  -d, --description string   Description of the machine identity
  -h, --help                 help for create
  -p, --permissions string   Permissions of the machine identity
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra machines list`

List machines

```
infra machines list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra machines remove`

Remove a machine identity

```
infra machines remove MACHINE [flags]
```

### Options

```
  -h, --help   help for remove
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra import`

Import an Infra server configuration

```
infra import FILE [flags]
```

### Options

```
  -h, --help      help for import
      --replace   replace any existing configuration
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra server`

Start Infra server

```
infra server [flags]
```

### Options

```
      --access-key string                   Access key (secret)
      --admin-access-key string             Admin access key (secret)
  -f, --config-file string                  Server configuration file
      --db-encryption-key string            Database encryption key (default "$HOME/.infra/sqlite3.db.key")
      --db-encryption-key-provider string   Database encryption key provider (default "native")
      --db-file string                      Path to SQLite 3 database (default "$HOME/.infra/sqlite3.db")
      --db-host string                      Database host
      --db-name string                      Database name
      --db-parameters string                Database additional connection parameters
      --db-password string                  Database password (secret)
      --db-port int                         Database port
      --db-username string                  Database username
      --enable-crash-reporting              Enable crash reporting (default true)
      --enable-telemetry                    Enable telemetry (default true)
      --enable-ui                           Enable Infra server UI
  -h, --help                                help for server
  -d, --session-duration duration           User session duration (default 12h0m0s)
      --tls-cache string                    Directory to cache TLS certificates (default "$HOME/.infra/cache")
      --ui-proxy-url string                 Proxy upstream UI requests to this url
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

## `infra engine`

Start Infra Engine

```
infra engine [flags]
```

### Options

```
  -a, --access-key string    Infra access key (use file:// to load from a file)
  -f, --config-file string   Engine config file
  -h, --help                 help for engine
  -n, --name string          Destination name
  -s, --server string        Infra server hostname
      --skip-tls-verify      Skip verifying server TLS certificates
      --tls-cache string     Directory to cache TLS certificates (default "$HOME/.infra/cache")
```

### Options inherited from parent commands

```
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
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
      --log-level string   Set the log level. One of error, warn, info, or debug (default "info")
```

