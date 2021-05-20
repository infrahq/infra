# CLI Reference

* [Install](#install)
* [Overview](#introduction)
    * [Global flags](#global-flags)
    * [Admin shell](#admin-shell)
* [Commands](#commands)
    * [`infra login`](#infra-login)
    * [`infra users list`](#infra-users-list)
    * [`infra users create`](#infra-users-create)
    * [`infra users delete`](#infra-users-delete)
    * [`infra users inspect` (Coming Soon)](#infra-users-inspect-coming-soon)
    * [`infra tokens list`](#infra-users-list)
    * [`infra tokens create`](#infra-tokens-create)
    * [`infra tokens delete`](#infra-tokens-delete)
    * [`infra engine`](#infra-engine)

## Install

See [Install Infra CLI](../README.md#install-infra-cli)

## Overview

### Admin mode

Running `infra` commands on the host machine or container of the Infra Engine automatically provides **admin** permissions.

This allows you to run commands without having to be logged in from an external client machine.

For example, using Kubernetes via `kubectl`:

```
kubectl -n infra exec -it infra-0 sh

# infra users list
USER            	EMAIL              	CREATED         PROVIDERS  	PERMISSION	  
usr_k3Egu0A9Jdah	bot@infrahq.com    	9 seconds ago	         	view      	
usr_cHHfCsZu3by7	michael@infrahq.com	6 hours ago  	okta     	view      	
usr_jojpIOMrBM6F	elon@infrahq.com   	6 hours ago  	okta     	view      	
usr_mBOjQx8RjC00	mark@infrahq.com   	6 hours ago  	okta     	view      	
usr_o7WreRsehzyn	tom@infrahq.com    	6 hours ago  	okta     	view      	
usr_uOQSaCwEDzYk	jeff@infrahq.com   	6 hours ago  	okta     	view      
```

### Global Flags

| Flag                 | Type       | Description                     |
| :----------------    | :-------   | :-----------------------------  |
| `--insecure, -i`     | `string`   | Trust self-signed certificates  |

## Commands

### `infra login`

#### Usage

```
$ infra login [flags] HOST
```

#### Flags

| Flag              | Type       | Description                    |
| :---------------- | :-------   | :----------------------------- |
| `--token, -t`     | `string`   | Token if logging in via token  |


#### Example (Okta)

```
$ infra login infra.acme.com

Choose a login method:
[x] Okta
[ ] GitLab
[ ] Token

✔ Logging in with Okta... success.
✔ Successfully logged in as michael@acme.com
✔ Kubeconfig updated
```

#### Example (Token)

```
$ infra login --token sk_ad9278ajdhs7odfso73hosi37fhso37l infra.acme.com
✔ Logging in with token... success.
✔ Successfully logged in as michael@acme.com
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
USER            	EMAIL              	CREATED         PROVIDERS  	PERMISSION	  
usr_k3Egu0A9Jdah	bot@infrahq.com    	9 seconds ago	         	view      	
usr_cHHfCsZu3by7	michael@infrahq.com	6 hours ago  	okta     	view      	
usr_jojpIOMrBM6F	elon@infrahq.com   	6 hours ago  	okta     	view      	
usr_mBOjQx8RjC00	mark@infrahq.com   	6 hours ago  	okta     	view      	
usr_o7WreRsehzyn	tom@infrahq.com    	6 hours ago  	okta     	view      	
usr_uOQSaCwEDzYk	jeff@infrahq.com   	6 hours ago  	okta     	view    
```

### `infra users create`

#### Usage

```
$ infra users create [flags] EMAIL
```

#### Flags

| Flag                   | Type       | Description                               |
| :-----------------     | :-------   | :---------------------------------------- |
| `--permission, -p`     | `string`   | Permission to grant user, default `view`  |


#### Example

```
$ infra users create michael@acme.com --permission view
usr_mgna7u291s012

Please share the following login with michael@acme.com:

infra login --token sk_Kc1dtcFazlIVFhkT2FsRjNaMmRGYVUxQk1kd18jdj10 31.58.101.169
```

### `infra users delete`

Delete a user

#### Usage

```
$ infra users delete USER
```

#### Example

```
$ infra users delete usr_mgna7u291s012
usr_mgna7u291s012
```

### `infra users inspect` (Coming Soon)

Inspect a user's permissions

#### Usage

```
$ infra users inspect USER
```

#### Example

```
$ infra user inspect usr_mgna7u291s012
INFRA RESOURCE                                                LIST  CREATE  UPDATE  DELETE
users                                                         ✔     ✔       ✔       ✔
groups                                                        ✔     ✔       ✔       ✔
providers                                                     ✔     ✔       ✔       ✔

KUBERNETES RESOURCE                                           LIST  CREATE  UPDATE  DELETE
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

HI

### `infra tokens list`

#### Usage

```
$ infra tokens list
```

#### Example

```
$ infra users list
USER            	EMAIL              	CREATED         PROVIDERS  	PERMISSION	  
usr_k3Egu0A9Jdah	bot@infrahq.com    	9 seconds ago	         	view      	
usr_cHHfCsZu3by7	michael@infrahq.com	6 hours ago  	okta     	view      	
usr_jojpIOMrBM6F	elon@infrahq.com   	6 hours ago  	okta     	view      	
usr_mBOjQx8RjC00	mark@infrahq.com   	6 hours ago  	okta     	view      	
usr_o7WreRsehzyn	tom@infrahq.com    	6 hours ago  	okta     	view      	
usr_uOQSaCwEDzYk	jeff@infrahq.com   	6 hours ago  	okta     	view    
```

### `infra tokens create`

Create a token for a user

#### Usage

```
$ infra tokens create USER
```

#### Example

```
$ infra token create usr_k3Egu0A9Jdah
sk_GqwGycdQhW00maZ9HeuizGp3VJfEmods2ik70pmy8cZt
```

The user can now log in via:
```
$ infra login --token sk_GqwGycdQhW00maZ9HeuizGp3VJfEmods2ik70pmy8cZt <infra endpoint>
```

### `infra tokens delete`

Delete a token

#### Usage

```
$ infra tokens delete TOKEN
```

#### Example

```
$ infra tokens delete tk_jg08aj08s40w
tk_jg08aj08s40w
```


### `infra engine`

Starts the Infra Engine

#### Usage

```
$ infra engine [--config, -c]
```

#### Flags

| Flag               | Type       | Description                                                 |
| :----------------- | :-------   | :---------------------------------------------------------- |
| `--config, -c`     | `string`   | Location of `infra.yaml` [config file](./configuration.md)  |
| `--db`             | `string`   | Directory to store database, defaults to `~/.infra`         |

#### Example

```
$ infra engine --config ./infra.yaml
```
