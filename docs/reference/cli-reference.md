---
title: CLI Reference
position: 1
---
# CLI Reference

## Commands



### `infra login`

Login to Infra

#### Description

Login to Infra and start a background agent to keep local configuration up-to-date

```
infra login [SERVER] [flags]
```

#### Examples

```
# By default, login will prompt for all required information.
$ infra login

# Login to a specific server
$ infra login infraexampleserver.com

# Login with a specific identity provider
$ infra login --provider okta

# Login with an access key
$ export INFRA_ACCESS_KEY=1M4CWy9wF5.fAKeKEy5sMLH9ZZzAur0ZIjy
$ infra login
```

#### Options

```
      --key string                       Login with an access key
      --no-agent                         Skip starting the Infra agent in the background
      --non-interactive                  Disable all prompts for input
      --provider string                  Login with an identity provider
      --skip-tls-verify                  Skip verifying server TLS certificates
      --tls-trusted-cert filepath        TLS certificate or CA used by the server
      --tls-trusted-fingerprint string   SHA256 fingerprint of the server TLS certificate
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra logout`

Log out of Infra

#### Description

Log out of Infra
Note: [SERVER] and [--all] cannot be both specified. Choose either one or all servers.

```
infra logout [SERVER] [flags]
```

#### Examples

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

#### Options

```
      --all     logout of all servers
      --clear   clear from list of servers
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra list`

List accessible destinations

```
infra list [flags]
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra use`

Access a destination

```
infra use DESTINATION [flags]
```

#### Examples

```

# Use a Kubernetes context
$ infra use development

# Use a Kubernetes namespace context
$ infra use development.kube-system
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra destinations list`

List connected destinations

```
infra destinations list [flags]
```

#### Options

```
      --format string   Output format [json|yaml]
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra destinations remove`

Disconnect a destination

```
infra destinations remove DESTINATION [flags]
```

#### Examples

```
$ infra destinations remove docker-desktop
```

#### Options

```
      --force   Exit successfully even if destination does not exist
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra grants list`

List grants

```
infra grants list [flags]
```

#### Options

```
      --destination string   Filter by destination
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra grants add`

Grant a user or group access to a destination

```
infra grants add USER|GROUP DESTINATION [flags]
```

#### Examples

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

#### Options

```
      --force         Create grant even if requested user, destination, or role are unknown
  -g, --group         When set, creates a grant for a group instead of a user
      --role string   Type of access that the user or group will be given (default "connect")
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra grants remove`

Revoke a user or group's access to a destination

```
infra grants remove USER|GROUP DESTINATION [flags]
```

#### Examples

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

#### Options

```
      --force         Exit successfully even if grant does not exist
  -g, --group         Group to revoke access from
      --role string   Role to revoke
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra users add`

Create a user

#### Description

Create a user.

Note: A temporary password will be created. The user will be prompted to set a new password on first login.

```
infra users add USER [flags]
```

#### Examples

```
# Create a user
$ infra users add johndoe@example.com
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra users edit`

Update a user

```
infra users edit USER [flags]
```

#### Examples

```
# Set a new password for a user
$ infra users edit janedoe@example.com --password
```

#### Options

```
      --password   Set a new password, or if admin, set a temporary password for the user
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra users list`

List users

```
infra users list [flags]
```

#### Options

```
      --format string   Output format [json|yaml]
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra users remove`

Delete a user

```
infra users remove USER [flags]
```

#### Examples

```
# Delete a user
$ infra users remove janedoe@example.com
```

#### Options

```
      --force   Exit successfully even if user does not exist
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups add`

Create a group

```
infra groups add GROUP [flags]
```

#### Examples

```
# Create a group
$ infra groups add Engineering
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups adduser`

Add a user to a group

```
infra groups adduser USER GROUP [flags]
```

#### Examples

```
# Add a user to a group
$ infra groups adduser johndoe@example.com Engineering

```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups list`

List groups

```
infra groups list [flags]
```

#### Options

```
      --no-truncate     Do not truncate the list of users for each group
      --num-users int   The number of users to display in each group (default 8)
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups remove`

Delete a group

```
infra groups remove GROUP [flags]
```

#### Examples

```
# Delete a group
$ infra groups remove Engineering
```

#### Options

```
      --force   Exit successfully even if the group does not exist
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups removeuser`

Remove a user from a group

```
infra groups removeuser USER GROUP [flags]
```

#### Examples

```
# Remove a user from a group
$ infra groups removeuser johndoe@example.com Engineering

```

#### Options

```
      --force   Exit successfully even if the user or group does not exist
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra keys list`

List access keys

```
infra keys list [flags]
```

#### Options

```
      --show-expired   Show expired access keys
      --user string    The name of a user to list access keys for
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra keys add`

Create an access key

#### Description

Create an access key for a user or a connector.

```
infra keys add USER|connector [flags]
```

#### Examples

```

# Create an access key named 'example-key' for a user that expires in 12 hours
$ infra keys add user@example.com --ttl=12h --name example-key

# Create an access key to add a Kubernetes connection to Infra
$ infra keys add connector

```

#### Options

```
      --extension-deadline duration   A specified deadline that the access key must be used within to remain valid (default 720h0m0s)
      --name string                   The name of the access key
      --ttl duration                  The total time that the access key will be valid for (default 720h0m0s)
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra keys remove`

Delete an access key

```
infra keys remove KEY [flags]
```

#### Options

```
      --force   Exit successfully even if access key does not exist
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra providers list`

List connected identity providers

```
infra providers list [flags]
```

#### Options

```
      --format string   Output format [json|yaml]
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra providers add`

Connect an identity provider

#### Description

Add an identity provider for users to authenticate.
PROVIDER is a short unique name of the identity provider being added (eg. okta)

```
infra providers add PROVIDER [flags]
```

#### Examples

```
# Connect Okta to Infra
$ infra providers add okta --url example.okta.com --client-id 0oa3sz06o6do0muoW5d7 --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN --kind okta

# Connect Google to Infra with group sync
$ infra providers add google --url accounts.google.com --client-id 0oa3sz06o6do0muoW5d7 --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN --service-account-key ~/client-123.json --service-account-email hello@example.com --domain-admin admin@example.com --kind google
```

#### Options

```
      --client-id string               OIDC client ID
      --client-secret string           OIDC client secret
      --domain-admin string            The email of your Google workspace domain admin
      --kind string                    The identity provider kind. One of 'oidc, okta, azure, or google' (default "oidc")
      --service-account-email string   The email assigned to the Infra service client in Google
      --service-account-key filepath   The private key used to make authenticated requests to Google's API
      --url string                     Base URL of the domain of the OIDC identity provider (eg. acme.okta.com)
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra providers edit`

Update a provider

```
infra providers edit PROVIDER [flags]
```

#### Examples

```
# Set a new client secret for a connected provider
$ infra providers edit okta --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN

# Connect Google to Infra with group sync
$ infra providers edit google --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN --service-account-key ~/client-123.json --service-account-email hello@example.com --domain-admin admin@example.com

```

#### Options

```
      --client-secret string           Set a new client secret
      --domain-admin string            The email of your Google workspace domain admin
      --service-account-email string   The email assigned to the Infra service client in Google
      --service-account-key filepath   The private key used to make authenticated requests to Google's API
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra providers remove`

Disconnect an identity provider

```
infra providers remove PROVIDER [flags]
```

#### Examples

```
$ infra providers remove okta
```

#### Options

```
      --force   Exit successfully even if provider does not exist
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra info`

Display the info about the current session

```
infra info [flags]
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra version`

Display the Infra version

```
infra version [flags]
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra about`

Display information about Infra

```
infra about [flags]
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra completion`

Generate shell auto-completion for the CLI

#### Description

To load completions:

Bash:

  $ source <(infra completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ infra completion bash > /etc/bash_completion.d/infra
  # macOS:
  $ infra completion bash > /usr/local/etc/bash_completion.d/infra

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ infra completion zsh > "${fpath[1]}/_infra"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ infra completion fish | source

  # To load completions for each session, execute once:
  $ infra completion fish > ~/.config/fish/completions/infra.fish

PowerShell:

  PS> infra completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> infra completion powershell > infra.ps1
  # and source this file from your PowerShell profile.


```
infra completion
```

#### Options inherited from parent commands

```
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
