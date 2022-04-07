# CLI Reference

## Commands

* [infra login](#infra-login)
* [infra logout](#infra-logout)
* [infra list](#infra-list)
* [infra use](#infra-use)
* [infra destinations list](#infra-destinations-list)
* [infra destinations remove](#infra-destinations-remove)
* [infra grants list](#infra-grants-list)
* [infra grants add](#infra-grants-add)
* [infra grants remove](#infra-grants-remove)
* [infra identities add](#infra-identities-add)
* [infra identities edit](#infra-identities-edit)
* [infra identities list](#infra-identities-list)
* [infra identities remove](#infra-identities-remove)
* [infra keys list](#infra-keys-list)
* [infra keys add](#infra-keys-add)
* [infra keys remove](#infra-keys-remove)
* [infra providers list](#infra-providers-list)
* [infra providers add](#infra-providers-add)
* [infra providers remove](#infra-providers-remove)


## `infra login`

Login to Infra

```
infra login [SERVER] [flags]
```

### Examples

```

# By default, login will prompt for all required information.
$ infra login

# Login to a specified server
$ infra login SERVER
$ infra login --server SERVER

# Login with an access key
$ infra login --key KEY

# Login with a specified provider
$ infra login --provider NAME

# Use the '--non-interactive' flag to error out instead of prompting.

```

### Options

```
      --key string        Login with an access key
      --provider string   Login with an identity provider
      --server string     Infra server to login to
      --skip-tls-verify   Skip verifying server TLS certificates
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra logout`

Log out of Infra

```
infra logout [flags]
```

### Examples

```
$ infra logout
```

### Options

```
      --purge   remove Infra host from config
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra list`

List accessible destinations

```
infra list [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra use`

Access a destination

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

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra destinations list`

List connected destinations

```
infra destinations list [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra destinations remove`

Disconnect a destination

```
infra destinations remove DESTINATION [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra grants list`

List grants

```
infra grants list [DESTINATION] [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra grants add`

Grant access to a destination

```
infra grants add DESTINATION [flags]
```

### Examples

```

# Grant user admin access to a cluster
$ infra grants add -u suzie@acme.com -r admin kubernetes.production

# Grant group admin access to a namespace
$ infra grants add -g Engineering -r admin kubernetes.production.default

# Grant user admin access to infra itself
$ infra grants add -u admin@acme.com -r admin infra

```

### Options

```
  -g, --group string      Group to grant access to
  -m, --machine string    Machine to grant access to
  -p, --provider string   Provider from which to grant user access to
  -r, --role string       Role to grant
  -u, --user string       User to grant access to
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra grants remove`

Revoke access to a destination

```
infra grants remove DESTINATION [flags]
```

### Options

```
  -g, --group string      Group to revoke access from
  -m, --machine string    Machine to revoke access from
  -p, --provider string   Provider from which to revoke access from
  -r, --role string       Role to revoke
  -u, --user string       User to revoke access from
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra identities add`

Create an identity.

### Synopsis

Create a machine identity with NAME or a user identity with EMAIL.

NAME must only contain alphanumeric characters ('a-z', 'A-Z', '0-9') or the
special characters '-', '_', or '/' and has a maximum length of 256 characters.

EMAIL must contain a valid email address in the form of "<local>@<domain>".
		

```
infra identities add NAME|EMAIL [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra identities edit`

Update an identity

```
infra identities edit NAME [flags]
```

### Options

```
  -p, --password   Prompt to update a local user's password
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra identities list`

List all identities

```
infra identities list [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra identities remove`

Delete an identity

```
infra identities remove NAME [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra keys list`

List access keys

```
infra keys list [flags]
```

### Options

```
  -m, --machine string   The name of a machine to list access keys for
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra keys add`

Create an access key for authentication

```
infra keys add ACCESS_KEY_NAME MACHINE_NAME [flags]
```

### Examples

```

# Add an access key for the machine "bot" called "first-key" that expires in 12 hours and must be used every hour to remain valid
infra keys add first-key bot --ttl=12h --extension-deadline=1h

```

### Options

```
      --extension-deadline string   A specified deadline that an access key must be used within to remain valid, defaults to 30 days
      --ttl string                  The total time that an access key will be valid for, defaults to 30 days
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra keys remove`

Delete an access key

```
infra keys remove ACCESS_KEY_NAME [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra providers list`

List connected identity providers

```
infra providers list [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra providers add`

Connect an identity provider

### Synopsis


Add an identity provider for users to authenticate.

NAME: The name of the identity provider (e.g. okta)
URL: The base URL of the domain used to login with the identity provider (e.g. acme.okta.com)
CLIENT_ID: The Infra application OpenID Connect client ID
CLIENT_SECRET: The Infra application OpenID Connect client secret
		

```
infra providers add NAME URL CLIENT_ID CLIENT_SECRET [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra providers remove`

Disconnect an identity provider

```
infra providers remove PROVIDER [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

