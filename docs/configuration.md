# Configuring Infra

* [Example](#example)
* [ConfigMap Usage](#configmap-usage)
* [Reference](#reference)
  * [`sources`](#sources)
    * [`okta`](#okta)
  * [`users`](#users)
    * [`name`](#user)
    * [`roles`](#roles)

## Overview

For teams who require configuration to be stored in version control, Infra can be managed via a configuration file, `infra.yaml`.

## Create a configuration file

First, create a config file `infra.yaml`:

```
users:
  - name: admin@example.com
    roles:
      - name: admin
        kind: cluster-role
        clusters:
          - cluster-AAA
          - cluster-BBB
```

Then, apply it to the Infra registry:

```
helm upgrade infra infrahq/infra --set-file config=./infra.yaml --recreate-pods
```

## Full Example

```yaml
sources:
  - type: okta
    domain: acme.okta.com
    clientId: 0oapn0qwiQPiMIyR35d6
    clientSecret: infrahq/oktaClientSecret
    apiToken: infrahq/oktaClientSecret

users:
  - name: admin@example.com
    roles:
      - name: admin
        kind: cluster-role
        clusters:
          - cluster-AAA
          - cluster-BBB
  - name: developer@example.com
    roles:
      - name: writer
        kind: cluster-role
        clusters:
          - cluster-AAA
```

## Configuration Reference

### `sources`

A list of sources to sync and authenticate users from

### `users`

A list of users for which to specify roles.

#### `name`

`name` is a user's email or username

#### `roles`

`roles` is a list of role mappings to Kubernetes roles
