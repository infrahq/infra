
# Configuring Infra

* [Overview](#overview)
* [Create a configuration file](#create-a-configuration-file)
* [Full Example](#full-example)
* [Reference](#reference)
  * [`sources`](#sources)
  * [`groups`](#groups)
  * [`users`](#users)
  * [`roles`](#roles)
  * [`destinations`](#destinations)

## Overview

For teams who require configuration to be stored in version control, Infra can be managed via a configuration file, `infra.yaml`.

## Create a configuration file

First, create a config file `infra.yaml`:

```
sources:
  - type: okta
    domain: acme.okta.com
    clientId: 0oapn0qwiQPiMIyR35d6
    clientSecret: infra-registry-okta/clientSecret
    apiToken: infra-registry-okta/apiToken

groups:
  - name: administrators
    source: okta
    roles:
      - name: cluster-admin
        kind: cluster-role
        destinations:
          - name: my-first-destination
```

Then, apply it to the Infra Registry:

```
helm upgrade infra-registry infrahq/registry -n infrahq --set-file config=./infra.yaml
```

## Full Example

```yaml
sources:
  - type: okta
    domain: acme.okta.com
    clientId: 0oapn0qwiQPiMIyR35d6
    clientSecret: infra-registry-okta/clientSecret
    apiToken: infra-registry-okta/apiToken

groups:
  - name: administrators
    source: okta
    roles:
      - name: cluster-admin
        kind: cluster-role
        destinations:
          - name: my-first-destination

users:
  - email: manager@example.com
    roles:
      - name: view
        kind: cluster-role
        destinations:
          - name: my-first-destination
            namespaces: 
              - infrahq
              - development
          - name: cluster-BBB
  - email: developer@example.com
    roles:
      - name: writer
        kind: role
        destinations:
          - name: my-first-destination
            namespaces:
              - development
```

## Reference

### `sources`

A list of sources to sync and authenticate users from

### `groups`

A list of groups from identity providers for which to specify roles

### `users`

A list of users from identity providers for which to specify roles

### `roles`

`roles` is a list of role mappings to Kubernetes roles

### `destinations`

A list of Kubernetes clusters a role mapping applies to
