# Configuration Reference

* [Example](#example)
* [Reference](#reference)
  * [`url`](#url)
  * [`providers`](#providers)
    * [`okta`](#okta)
  * [`permissions`](#permissions)
    * [`user`](#user)
    * [`permission`](#permission)
    * [`namespace`](#namespace)

## Example

```yaml
providers:
  okta:
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: /var/run/infra/secrets/okta-client-secret
    api-token: /var/run/infra/secrets/okta-api-token

permissions:
  - user: admin@acme.com
    permission: admin
  - user: suzie@acme.com
    permission: edit
  - user: bob@acme.com
    permission: view
```

## Reference

### `providers`

#### `okta`

* `domain`: Okta domain
* `client-id`: Client ID for the Okta application
* `client-secret`: Path to file containing client secret for the Okta Application
* `api-token`: Path to file containing Okta API Token

Example:

```yaml
providers:
  okta:
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: /etc/secrets/infra/okta-client-secret
    api-token: /etc/secrets/infra/okta-api-token
```

### `users`

### Example

```yaml
permissions:
  - email: admin@acme.com
    permission: admin
  - email: jeff@acme.com
    permission: edit
    namespace: default
```

### `user`

`user` is a user's email

### `permission`

`permission` defines a permission level, giving users access to specific resources and tasks they need

| Permission | Description                        |
| :--------  | :------------------------------    |
| view       | View & list any resource           |
| edit       | Create, edit, delete any resource  |
| admin      | Full access                        |

### `namespace`

* `namespace` is a Kubernetes namespace to scope permissions for
