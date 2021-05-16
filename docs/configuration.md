# Configuration Reference

* [Example](#example)
* [Reference](#reference)
  * [`providers`](#providers)
    * [`okta`](#okta)
  * [`users`](#users)
    * [`email`](#email)
    * [`permission`](#permission)
    * [`namespace`](#namespace)

## Example

```yaml
providers:
  okta:
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: /etc/secrets/infra/okta-client-secret
    api-token: /etc/secrets/infra/okta-api-token

users:
  - email: admin@acme.com
    permission: admin
  - email: jeff@acme.com
    permission: edit
    namespace: default
```

## Reference

### `providers`

#### `okta`

* `domain`: Okta domain
* `client-id`: Client ID for the Okta application
* `client-secret`: Client Secret for the Okta application
* `api-token`: Read-only Okta API Token

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
users:
  - email: admin@acme.com
    permission: admin
  - email: jeff@acme.com
    permission: edit
    namespace: default
```

### `email`

`email` is a user's email

### `permission`

`permission` defines a permission level, giving users access to specific resources and tasks they need

| Permission | Description                     |
| :--------  | :-------------------------      |
| view       | View & list any resource        |
| edit       | View & list any resource        |
| admin      | Full acces                      |

### `namespace`

* `namespace` is a Kubernetes namespace to scope permissions for
