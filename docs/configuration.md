# Configuring Infra

* [Example](#example)
* [ConfigMap Usage](#configmap-usage)
* [Reference](#reference)
  * [`sources`](#sources)
    * [`okta`](#okta)
  * [`users`](#roles)
    * [`name`](#user)
    * [`roles`](#role)

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
helm upgrade infra --set config=./infra.yaml --recreate-pods
```

## Full Example

```yaml
sources:
  - kind: okta
    domain: acme.okta.com
    clientId: 0oapn0qwiQPiMIyR35d6
    clientSecret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
    apiToken: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd

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

#### `name`

`name` is a user's email or username

#### `roles`

`roles` is a list of role mappings to Kubernetes roles

