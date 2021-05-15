# Token

## Contents

* [Setup](#setup)
* [Usage](#usage)
    * [Create a user](#log-in-with-okta)
    * [List Okta users](#list-okta-users)

## Setup

If you don't have any users in Infra Engine, you can add users via `kubectl`:

```
$ kubectl exec -n infra infra-0 -- infra users create admin@acme.com --permission view

User admin@acme.com added. Please share the following command with them so they can log in:

infra login --token sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7
```

## Usage

### Log in with Token

```
$ infra login --token sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7 infra.acme.com
✔ Logging in with Okta... success
✔ Logged in as michael@acme.com
✔ Kubeconfig updated
```

### Create a user

```
$ infra users create bob@acme.com --permission view

User bob@acme.com added. Please share the following command with them so they can log in:

infra login --token sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7
```

### Delete a user

```
$ infra users delete usr_vfZjSZctMptn
usr_vfZjSZctMptn
```

### List users

```
$ infra users ls
USER ID         	PROVIDERS	EMAIL             CREATED     	  PERMISSION
usr_vfZjSZctMptn	token     	bob@acme.com      2 minutes ago   view
```