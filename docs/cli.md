---
title: CLI Reference
---
# CLI Reference

## Commands

### `infra login`

Login to Infra

```bash
infra login [SERVER] [flags]
```

#### Examples

```bash
# Login
infra login example.infrahq.com

# Login with username and password (prompt for password)
infra login example.infrahq.com --user user@example.com

# Login with access key
export INFRA_SERVER=example.infrahq.com
export INFRA_ACCESS_KEY=2vrEbqFEUr.jtTlxkgYdvghJNdEa8YoUxN0
infra login example.infrahq.com --user user@example.com

# Login with username and password
export INFRA_SERVER=example.infrahq.com
export INFRA_USER=user@example.com
export INFRA_PASSWORD=p4ssw0rd
infra login
```

#### Options

```console
      --key string                       Login with an access key
      --no-agent                         Skip starting the Infra agent in the background
      --non-interactive                  Disable all prompts for input
      --skip-tls-verify                  Skip verifying server TLS certificates
      --tls-trusted-cert filepath        TLS certificate or CA used by the server
      --tls-trusted-fingerprint string   SHA256 fingerprint of the server TLS certificate
      --user string                      User email
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra logout`

Log out of Infra

#### Description

Log out of Infra
Note: [SERVER] and [--all] cannot be both specified. Choose either one or all servers.

```bash
infra logout [SERVER] [flags]
```

#### Examples

```bash
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

```console
      --all     logout of all servers
      --clear   clear from list of servers
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra list`

List accessible destinations

```bash
infra list [flags]
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra use`

Access a destination

```bash
infra use DESTINATION [flags]
```

#### Examples

```bash

# Use a Kubernetes context
$ infra use development

# Use a Kubernetes namespace context
$ infra use development.kube-system
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra destinations list`

List connected destinations

```bash
infra destinations list [flags]
```

#### Options

```console
      --format string   Output format [json|yaml]
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra destinations remove`

Disconnect a destination

```bash
infra destinations remove DESTINATION [flags]
```

#### Examples

```bash
$ infra destinations remove docker-desktop
```

#### Options

```console
      --force   Exit successfully even if destination does not exist
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra grants list`

List grants

```bash
infra grants list [flags]
```

#### Options

```console
      --destination string   Filter by destination
      --group string         Filter by group name or id
      --inherited            Include grants a user inherited through a group
      --resource string      Filter by resource
      --role string          Filter by user role
      --user string          Filter by user name or id
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra grants add`

Grant a user or group access to a destination

```bash
infra grants add USER|GROUP DESTINATION [flags]
```

#### Examples

```bash
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

```console
      --force         Create grant even if requested user, destination, or role are unknown
  -g, --group         When set, creates a grant for a group instead of a user
      --role string   Type of access that the user or group will be given (default "connect")
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra grants remove`

Revoke a user or group's access to a destination

```bash
infra grants remove USER|GROUP DESTINATION [flags]
```

#### Examples

```bash
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

```console
      --force         Exit successfully even if grant does not exist
  -g, --group         Group to revoke access from
      --role string   Role to revoke
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra users add`

Create a user

#### Description

Create a user.

Note: A temporary password will be created. The user will be prompted to set a new password on first login.

```bash
infra users add USER [flags]
```

#### Examples

```bash
# Create a user
$ infra users add johndoe@example.com
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra users edit`

Update a user

```bash
infra users edit USER [flags]
```

#### Examples

```bash
# Set a new password for a user
$ infra users edit janedoe@example.com --password
```

#### Options

```console
      --password   Set a new password, or if admin, set a temporary password for the user
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra users list`

List users

```bash
infra users list [flags]
```

#### Options

```console
      --format string   Output format [json|yaml]
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra users remove`

Delete a user

```bash
infra users remove USER [flags]
```

#### Examples

```bash
# Delete a user
$ infra users remove janedoe@example.com
```

#### Options

```console
      --force   Exit successfully even if user does not exist
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups add`

Create a group

```bash
infra groups add GROUP [flags]
```

#### Examples

```bash
# Create a group
$ infra groups add Engineering
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups adduser`

Add a user to a group

```bash
infra groups adduser USER GROUP [flags]
```

#### Examples

```bash
# Add a user to a group
$ infra groups adduser johndoe@example.com Engineering

```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups list`

List groups

```bash
infra groups list [flags]
```

#### Options

```console
      --no-truncate     Do not truncate the list of users for each group
      --num-users int   The number of users to display in each group (default 8)
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups remove`

Delete a group

```bash
infra groups remove GROUP [flags]
```

#### Examples

```bash
# Delete a group
$ infra groups remove Engineering
```

#### Options

```console
      --force   Exit successfully even if the group does not exist
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra groups removeuser`

Remove a user from a group

```bash
infra groups removeuser USER GROUP [flags]
```

#### Examples

```bash
# Remove a user from a group
$ infra groups removeuser johndoe@example.com Engineering

```

#### Options

```console
      --force   Exit successfully even if the user or group does not exist
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra keys list`

List access keys

```bash
infra keys list [flags]
```

#### Options

```console
      --all            Show keys for all users
      --show-expired   Show expired access keys
      --user string    The name of a user to list access keys for
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra keys add`

Create an access key

#### Description

Create an access key for a user or a connector.

```bash
infra keys add [flags]
```

#### Examples

```bash

# Create an access key named 'example-key' for a user that expires in 12 hours
$ infra keys add --expiry=12h --name example-key

# Create an access key to add a Kubernetes connection to Infra
$ infra keys add --connector

# Set an environment variable with the newly created access key
$ MY_ACCESS_KEY=$(infra keys add -q --name my-key)

```

#### Options

```console
      --connector                     Create the key for the connector
      --expiry duration               The total time that the access key will be valid for (default 8760h0m0s)
      --inactivity-timeout duration   A specified deadline that the access key must be used within to remain valid (default 720h0m0s)
      --name string                   The name of the access key
  -q, --quiet                         Only display the access key
      --user string                   The name of the user who will own the key
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra keys remove`

Delete an access key

```bash
infra keys remove KEY [flags]
```

#### Options

```console
      --force         Exit successfully even if access key does not exist
      --user string   The name of the user who owns the key
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra providers list`

List connected identity providers

```bash
infra providers list [flags]
```

#### Options

```console
      --format string   Output format [json|yaml]
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra providers add`

Connect an identity provider

#### Description

Add an identity provider for users to authenticate.
PROVIDER is a short unique name of the identity provider being added (eg. okta)

```bash
infra providers add PROVIDER [flags]
```

#### Examples

```bash
# Connect Okta to Infra
$ infra providers add okta --url example.okta.com --client-id 0oa3sz06o6do0muoW5d7 --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN --kind okta

# Connect Google to Infra with group sync
$ infra providers add google --url accounts.google.com --client-id 0oa3sz06o6do0muoW5d7 --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN --service-account-key ~/client-123.json --workspace-domain-admin admin@example.com --kind google
```

#### Options

```console
      --client-id string                OIDC client ID
      --client-secret string            OIDC client secret
      --kind string                     The identity provider kind. One of 'oidc, okta, azure, or google' (default "oidc")
      --scim                            Create an access key for SCIM provisioning
      --service-account-email string    The email assigned to the Infra service client in Google
      --service-account-key filepath    The private key used to make authenticated requests to Google's API, can be a file or the key string directly
      --url string                      Base URL of the domain of the OIDC identity provider (eg. acme.okta.com)
      --workspace-domain-admin string   The email of your Google Workspace domain admin
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra providers edit`

Update a provider

```bash
infra providers edit PROVIDER [flags]
```

#### Examples

```bash
# Set a new client secret for a connected provider
$ infra providers edit okta --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN

# Connect Google to Infra with group sync
$ infra providers edit google --client-secret VT_oXtkEDaT7UFY-C3DSRWYb00qyKZ1K1VCq7YzN --service-account-key ~/client-123.json --service-account-email hello@example.com --workspace-domain-admin admin@example.com

```

#### Options

```console
      --client-secret string            Set a new client secret
      --scim                            Create a new access key for SCIM provisioning
      --service-account-email string    The email assigned to the Infra service client in Google
      --service-account-key filepath    The private key used to make authenticated requests to Google's API
      --workspace-domain-admin string   The email of your Google workspace domain admin
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra providers remove`

Disconnect an identity provider

```bash
infra providers remove PROVIDER [flags]
```

#### Examples

```bash
$ infra providers remove okta
```

#### Options

```console
      --force   Exit successfully even if provider does not exist
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra info`

Display the info about the current session

```bash
infra info [flags]
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra version`

Display the Infra version

```bash
infra version [flags]
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra about`

Display information about Infra

```bash
infra about [flags]
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
### `infra completion`

Generate shell auto-completion for the CLI

#### Description

To load completions:

##### Bash:

`$ source <(infra completion bash)`

To load completions for each session, execute once:
* Linux:
  `$ infra completion bash > /etc/bash_completion.d/infra`
* macOS:
  `$ infra completion bash > /usr/local/etc/bash_completion.d/infra`

##### Zsh:

If shell completion is not already enabled in your environment, you will need to enable it. You can execute the following once:
`$ echo "autoload -U compinit; compinit" >> ~/.zshrc`

To load completions for each session, execute once:
`$ infra completion zsh > "${fpath[1]}/_infra"`

You will need to start a new shell for this setup to take effect.

##### fish:

`$ infra completion fish | source`

To load completions for each session, execute once:
`$ infra completion fish > ~/.config/fish/completions/infra.fish`

##### PowerShell:

`PS> infra completion powershell | Out-String | Invoke-Expression`

To load completions for every new session, run:
`PS> infra completion powershell > infra.ps1`
and source this file from your PowerShell profile.


```bash
infra completion
```

**Additional options**

```console
      --help               Display help
      --log-level string   Show logs when running the command [error, warn, info, debug] (default "info")
```
