# CLI Reference

* [Install](#install)
* [Overview](#introduction)
    * [Global flags](#global-flags)
    * [Admin mode](#admin-mode)
* [Commands](#commands)
    * [`infra login`](#infra-login)
    * [`infra users list`](#infra-users-list)
    * [`infra users create`](#infra-users-create)
    * [`infra users delete`](#infra-users-delete)
    * [`infra users inspect` (Coming Soon)](#infra-users-inspect-coming-soon)
    * [`infra server`](#infra-server)

## Install

See [Install Infra CLI](../README.md#install-infra-cli)

## Overview

### Admin mode

Running `infra` commands on the host machine or container of the Infra automatically provides **admin** permissions.

This allows you to run commands without having to be logged in from an external client machine.

For example, using Kubernetes via `kubectl`:

```
kubectl -n infra exec -it infra-0 sh

# infra users list
EMAIL              	PROVIDER	PERMISSION	CREATED    
admin@example.com  	infra   	view      	3 days ago	
tom@example.com    	okta    	view      	3 days ago	
jeff@example.com*  	okta    	admin     	3 days ago	
michael@example.com	okta    	view      	3 days ago	
elon@example.com   	okta    	view      	3 days ago	
mark@example.com   	okta    	view      	3 days ago
```

### Global Flags

| Flag                 | Type       | Description                       |
| :----------------    | :-------   | :-----------------------------    |
| `--insecure, -i`     | `string`   | Disable TLS verification          |
| `--host, -H`         | `string`   | Infra server host to connect to   |
| `--token, -t`        | `string`   | Authentication token              |

## Commands

### `infra login`

#### Usage

```
$ infra login [flags] HOST
```

#### Flags

None

#### Example (Username & Password)

```
$ infra login infra.example.com
? Choose a login provider  [Use arrows to move, type to filter]
  Okta [example.okta.com]
> Username & password
? Email user@example.com
? Password **********
✔ Logging in with username username & password... success.
✔ Logged in...
✔ Kubeconfig updated
```


#### Example (Okta)

```
$ infra login infra.acme.com
? Choose a login provider  [Use arrows to move, type to filter]
> Okta [example.okta.com]
  Username & password
✔ Logging in with Okta... success.
✔ Logged in...
✔ Kubeconfig updated
```

### `infra users list`

#### Usage

```
$ infra users list
```

#### Example

```
$ infra users list
EMAIL              	PROVIDER	PERMISSION	CREATED    
admin@example.com  	infra   	view      	3 days ago	
tom@example.com    	okta    	view      	3 days ago	
jeff@example.com*  	okta    	admin     	3 days ago	
michael@example.com	okta    	view      	3 days ago	
elon@example.com   	okta    	view      	3 days ago	
mark@example.com   	okta    	view      	3 days ago 
```

### `infra users create`

#### Usage

```
$ infra users create [flags] EMAIL PASSWORD
```

#### Flags

None

#### Example

```
$ infra users create michael@acme.com passw0rd
```

### `infra users delete`

Delete a user

#### Usage

```
$ infra users delete USER
```

#### Example

```
$ infra users delete michael@acme.com
```

### `infra users inspect` (Coming Soon)

Inspect a user's permissions

#### Usage

```
$ infra users inspect USER
```

#### Example

```
$ infra user inspect michael@acme.com
RESOURCE                                                      LIST  CREATE  UPDATE  DELETE
daemonsets.apps                                               ✔     ✔       ✔       ✔
daemonsets.extensions                                         ✔     ✔       ✔       ✔
deployments.apps                                              ✔     ✔       ✔       ✔
deployments.extensions                                        ✔     ✔       ✔       ✔
endpoints                                                     ✔     ✔       ✔       ✔
events                                                        ✔     ✔       ✔       ✔
events.events.k8s.io                                          ✔     ✔       ✔       ✔
pods                                                          ✔     ✔       ✔       ✔
pods.metrics.k8s.io                                           ✔                     
podsecuritypolicies.extensions                                ✔     ✔       ✔       ✔
podsecuritypolicies.policy                                    ✔     ✔       ✔       ✔
replicasets.apps                                              ✔     ✔       ✔       ✔
replicasets.extensions                                        ✔     ✔       ✔       ✔
replicationcontrollers                                        ✔     ✔       ✔       ✔
resourcequotas                                                ✔     ✔       ✔       ✔
rolebindings.rbac.authorization.k8s.io                        ✔     ✔       ✔       ✔
roles.rbac.authorization.k8s.io                               ✔     ✔       ✔       ✔
runtimeclasses.node.k8s.io                                    ✔     ✔       ✔       ✔
secrets                                                       ✔     ✔       ✔       ✔ 
selfsubjectaccessreviews.authorization.k8s.io                       ✔               
selfsubjectrulesreviews.authorization.k8s.io                        ✔               
serviceaccounts                                               ✔     ✔       ✔       ✔
services                                                      ✔     ✔       ✔       ✔
statefulsets.apps                                             ✔     ✔       ✔       ✔
storageclasses.storage.k8s.io                                 ✔     ✔       ✔       ✔
subjectaccessreviews.authorization.k8s.io                           ✔               
tokenreviews.authentication.k8s.io                                  ✔               
validatingwebhookconfigurations.admissionregistration.k8s.io  ✔     ✔       ✔       ✔
volumeattachments.storage.k8s.io                              ✔     ✔       ✔       ✔
```

### `infra server`

Starts the Infra Server

#### Usage

```
$ infra server [--config, -c]
```

#### Flags

| Flag               | Type       | Description                                                       |
| :----------------- | :-------   | :----------------------------------------------------------       |
| `--config, -c`     | `string`   | Location of `infra.yaml` [config file](./configuration.md)        |
| `--db`             | `string`   | Directory to store database, defaults to `~/.infra`               |
| `--tls-cache       | `string`   | Directory to cache tls certificates, defaults to `~/.infra/cache` |
| `--ui`             | `string`   | Enable UI                                                         |
| `--ui-proxy`       | `string`   | Proxy to ui requests to development server                        |

#### Example

```
$ infra server --config ./infra.yaml
```
