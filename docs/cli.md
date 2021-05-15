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
    * [`infra engine`](#infra-engine)

## Install

```bash
# macOS
$ curl --url "https://github.com/infrahq/infra/releases/download/latest/infra-darwin-$(uname -m)" --output /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Linux
$ curl --url "https://github.com/infrahq/infra/releases/download/latest/infra-linux-$(uname -m)" --output /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Windows 10
$ curl.exe --url "https://github.com/infrahq/infra/releases/download/latest/infra-windows-amd64.exe" --output infra.exe
```

## Overview

### Admin shell

To use the Infra CLI without logging in locally, you can open an admin shell to Infra Engine:

#### Example: Kubernetes

```
kubectl -n infra exec -it infra-0 sh

# infra users
ID                      NAME                  PROVIDER           CREATED                PERMISSION
usr_180jhsxjnxui1       jeff@acme.com         okta               2 minutes ago          admin
usr_mgna7u291s012       michael@acme.com      okta               2 minutes ago          view
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
ID                      NAME                  PROVIDER           CREATED                PERMISSION
usr_180jhsxjnxui1       jeff@acme.com         okta               2 minutes ago          admin
usr_mgna7u291s012       michael@acme.com      okta               2 minutes ago          view
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