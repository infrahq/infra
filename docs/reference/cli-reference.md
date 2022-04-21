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

### Examples

```
$ infra destinations remove kubernetes.docker-desktop
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

Grant an identity access to a destination

```
infra grants add IDENTITY DESTINATION [flags]
```

### Examples

```
# Grant an identity access to a destination
$ infra grants add johndoe@example.com kubernetes.docker-desktop 
$ infra grants add machine-a kubernetes.docker-desktop

# Grant a group access to a destination 
$ infra grants add group-a kubernetes.staging --group

# Grant access with fine-grained permissions
$ infra grants add johndoe@example.com kubernetes.staging --role viewer

# Assign a user a role within Infra
$ infra grants add johndoe@example.com infra --role admin

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

Revoke an identity's access from a destination

```
infra grants remove IDENTITY DESTINATION [flags]
```

### Examples

```
# Remove all grants of an identity in a destination
$ infra grants remove janedoe@example.com kubernetes.docker-desktop 
$ infra grants remove machine-a kubernetes.docker-desktop

# Remove all grants of a group in a destination
$ infra grants remove group-a kubernetes.staging --group

# Remove a specific grant 
$ infra grants remove janedoe@example.com kubernetes.staging --role viewer

# Remove access to infra 
$ infra grants remove janedoe@example.com infra --role admin

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

Create an identity.

If a valid email is detected, a user identity is created. 
If a username is detected, a machine identity is created.

A new user identity must change their one time password before further usage.

```
infra identities add IDENTITY [flags]
```

### Examples

```
# Create a local user
$ infra identities add johndoe@example.com

# Create a machine
$ infra identities add machineA
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

### Examples

```
# Set a new one time password for a local user
$ infra identities edit janedoe@example.com --password
```

### Options

```
  -p, --password   Set a new one time password
```

### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
      --non-interactive    Disable all prompts for input
```

## `infra identities list`

List identities

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
infra identities remove IDENTITY [flags]
```

### Examples

```
# Delete a local user
$ infra identities remove janedoe@example.com

# Delete a machine
$ infra identities remove machineA
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

Create an access key

### Synopsis

Create an access key. Only machine identities are supported at this time.

```
infra keys add KEY IDENTITY [flags]
```

### Examples

```

# Create an access key named 'example-key' that expires in 12 hrs
$ infra keys add example-key machine-a --ttl=12h

```

### Options

```
      --extension-deadline duration   A specified deadline that an access key must be used within to remain valid (default 720h0m0s)
      --ttl duration                  The total time that an access key will be valid for (default 720h0m0s)
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
infra keys remove KEY [flags]
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
      --non-interactive    Disable all prompts for input
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

