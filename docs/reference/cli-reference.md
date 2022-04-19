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
* [infra about](#infra-about)


## `infra login`

Login to Infra

```
infra login [SERVER] [flags]
```

### Examples

```
# By default, login will prompt for all required information.
$ infra login

# Login to a specific server
$ infra login infraexampleserver.com

# Login with a specific identity provider
$ infra login --provider okta

# Login with an access key
$ infra login --key 1M4CWy9wF5.fAKeKEy5sMLH9ZZzAur0ZIjy
```

### Options

```
      --key string        Login with an access key
      --provider string   Login with an identity provider
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

### Synopsis

Log out of Infra
Note: [SERVER] and [--all] cannot be both specified. Choose either one or all servers.

```
infra logout [SERVER] [flags]
```

### Examples

```
# Log out of current server
$ infra logout
		
# Log out of a specific server
$ infra logout infraexampleserver.com
		
# Logout of all servers
$ infra logout --all 
		
# Log out of current server and clear from list 
$ infra logout --clear
		
# Log out of a specific server and clear from list
$ infra logout infraexampleserver.com --clear 
		
# Logout and clear list of all servers 
$ infra logout --all --clear
```

### Options

```
      --all     logout of all servers
      --clear   clear from list of servers
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

# Use a Kubernetes context
$ infra use development

# Use a Kubernetes namespace context
$ infra use development.kube-system
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
infra grants list [flags]
```

### Options

```
      --destination string   Filter by destination
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra grants add`

Grant access to a destination

### Synopsis

Grant one or more identities access to a destination. 

IDENTITY is the subject that is being given access.
DESTINATION is what the identity will gain access to. 

Use [--role] if further fine grained permissions are needed. If not specified, user will gain the permission 'connect' to the destination. 
$ infra grants add ... -role admin ...

Use [--group] or [-g] if identity is of type group. 
$ infra grants add devGroup -group ...
$ infra grants add devGroup -g ...

For full documentation on grants with more examples, see: 
  https://github.com/infrahq/infra/blob/main/docs/guides


```
infra grants add IDENTITY DESTINATION [flags]
```

### Options

```
  -g, --group         Required if identity is of type 'group'
      --role string   Type of access that identity will be given (default "connect")
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra grants remove`

Revoke access to a destination

### Synopsis

Revokes access that user has to the destination.

IDENTITY is one that was being given access.
DESTINATION is what the identity will lose access to. 

Use [--role] to specify the exact grant being deleted. 
If not specified, it will revoke all roles for that user within the destination. 

Use [--group] or [-g] if identity is of type group. 
$ infra grants remove devGroup -g ...


```
infra grants remove IDENTITY DESTINATION [flags]
```

### Options

```
  -g, --group         Group to revoke access from
      --role string   Role to revoke
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

EMAIL must contain a valid email address in the form of "local@domain".
		

```
infra identities add IDENTITY [flags]
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
infra identities edit IDENTITY [flags]
```

### Options

```
  -p, --password   Update password field
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

# Create an access key for the machine "bot" called "first-key" that expires in 12 hours and must be used every hour to remain valid
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

PROVIDER is a short unique name of the identity provider bieng added (eg. okta) 
		

```
infra providers add PROVIDER [flags]
```

### Options

```
      --client-id string       OIDC client ID
      --client-secret string   OIDC client secret
      --url string             Base URL of the domain of the OIDC identity provider (eg. acme.okta.com)
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

## `infra about`

Display information about Infra

```
infra about [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

