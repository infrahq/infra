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
* [infra users add](#infra-users-add)
* [infra users edit](#infra-users-edit)
* [infra users list](#infra-users-list)
* [infra users remove](#infra-users-remove)
* [infra keys list](#infra-keys-list)
* [infra keys add](#infra-keys-add)
* [infra keys remove](#infra-keys-remove)
* [infra providers list](#infra-providers-list)
* [infra providers add](#infra-providers-add)
* [infra providers remove](#infra-providers-remove)
* [infra info](#infra-info)
* [infra version](#infra-version)
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
      --non-interactive   Disable all prompts for input
      --provider string   Login with an identity provider
      --skip-tls-verify   Skip verifying server TLS certificates
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
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
```

## `infra destinations list`

List connected destinations

```
infra destinations list [flags]
```

### Options

```
      --format string   Output format [text, json] (default "text")
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra destinations remove`

Disconnect a destination

```
infra destinations remove DESTINATION [flags]
```

### Examples

```
$ infra destinations remove docker-desktop
```

### Options

```
      --force   Exit successfully even if destination does not exist
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
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
```

## `infra grants add`

Grant a user or group access to a destination

```
infra grants add USER|GROUP DESTINATION [flags]
```

### Examples

```
# Grant a user access to a destination
$ infra grants add johndoe@example.com docker-desktop

# Grant a group access to a destination
$ infra grants add group-a staging --group

# Grant access with fine-grained permissions
$ infra grants add johndoe@example.com staging --role viewer

# Assign a user a role within Infra
$ infra grants add johndoe@example.com infra --role admin

```

### Options

```
      --force         Create grant even if requested user, destination, or role are unknown
  -g, --group         When set, creates a grant for a group instead of a user
      --role string   Type of access that the user or group will be given (default "connect")
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra grants remove`

Revoke a user or group's access to a destination

```
infra grants remove USER|GROUP DESTINATION [flags]
```

### Examples

```
# Remove all grants of a user in a destination
$ infra grants remove janedoe@example.com docker-desktop

# Remove all grants of a group in a destination
$ infra grants remove group-a staging --group

# Remove a specific grant
$ infra grants remove janedoe@example.com staging --role viewer

# Remove adminaccess to infra
$ infra grants remove janedoe@example.com infra --role admin

```

### Options

```
      --force         Exit successfully even if grant does not exist
  -g, --group         Group to revoke access from
      --role string   Role to revoke
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra users add`

Create a user.

### Synopsis

Create a user.

Note: A new user must change their one time password before further usage.

```
infra users add USER [flags]
```

### Examples

```
# Create a user
$ infra users add johndoe@example.com
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra users edit`

Update a user

```
infra users edit USER [flags]
```

### Examples

```
# Set a new one time password for a user
$ infra users edit janedoe@example.com --password
```

### Options

```
      --password   Set a new one time password
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra users list`

List users

```
infra users list [flags]
```

### Options

```
      --format string   Output format [text, json] (default "text")
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra users remove`

Delete a user

```
infra users remove USER [flags]
```

### Examples

```
# Delete a user
$ infra users remove janedoe@example.com
```

### Options

```
      --force   Exit successfully even if user does not exist
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra keys list`

List access keys

```
infra keys list [flags]
```

### Options

```
      --user string   The name of a user to list access keys for
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra keys add`

Create an access key

### Synopsis

Create an access key for a user or a connector.

```
infra keys add USER|connector [flags]
```

### Examples

```

# Create an access key named 'example-key' for a user that expires in 12 hours
$ infra keys add example-key user@example.com --ttl=12h

# Create an access key to add a Kubernetes connection to Infra
$ infra keys add connector

```

### Options

```
      --extension-deadline duration   A specified deadline that the access key must be used within to remain valid (default 720h0m0s)
      --name string                   The name of the access key
      --ttl duration                  The total time that the access key will be valid for (default 720h0m0s)
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra keys remove`

Delete an access key

```
infra keys remove KEY [flags]
```

### Options

```
      --force   Exit successfully even if access key does not exist
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra providers list`

List connected identity providers

```
infra providers list [flags]
```

### Options

```
      --format string   Output format [text, json] (default "text")
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra providers add`

Connect an identity provider

### Synopsis

Add an identity provider for users to authenticate.
PROVIDER is a short unique name of the identity provider being added (eg. okta)

```
infra providers add PROVIDER [flags]
```

### Examples

```
# Connect okta to infra
$ infra providers add okta --url example.okta.com --client-id 0oa3sz06o6do0muoW5d7 --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN
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
```

## `infra providers remove`

Disconnect an identity provider

```
infra providers remove PROVIDER [flags]
```

### Examples

```
$ infra providers remove okta
```

### Options

```
      --force   Exit successfully even if provider does not exist
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra info`

Display the info about the current session

```
infra info [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```

## `infra version`

Display the Infra version

```
infra version [flags]
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
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
```

