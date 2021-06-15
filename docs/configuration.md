# Configuration Reference

* [Example](#example)
* [Reference](#reference)
  * [`providers`](#providers)
    * [`okta`](#okta)
  * [`grants`](#grants)
    * [`user`](#user)
    * [`resource`](#resource)

## Example

```yaml
providers:
  - kind: infra
  - kind: okta
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
    api-token: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd

grants:
  - user: admin@example.com
    resource: production
    role: admin
```

## Reference

### `providers`

#### `okta`

* `domain`: Okta domain
* `client-id`: Client ID for the Okta application
* `client-secret`: Client Secret for the Okta application
* `api-token`: Okta API Token

Example:

```yaml
providers:
  - kind: okta
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
    api-token: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd
```

### `grants`

### Example

```yaml
grants:
  - user: admin@infrahq.com
    role: infra.owner
  - user: jeff@infrahq.com
    role: edit
  - user: michael@infrahq.com
    role: view
  - user: michael@infrahq.com
    role: infra.owner
```

### `user`

`user` is a user's email

### `role`

`role` defines a permission level, giving users access to specific resources and tasks they need

| Role       | Description                        |
| :--------  | :------------------------------    |
| view       | Read-only for most resources       |
| edit       | Read & write most resources        |
| admin      | Read & write any resource          |

