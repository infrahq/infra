# Configuration Reference

* [Example](#example)
* [Reference](#reference)
  * [`providers`](#providers)
    * [`okta`](#okta)
  * [`permissions`](#permissions)
    * [`name`](#permission)
    * [`users`](#user)

## Example

```yaml
providers:
  - kind: okta
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
    api-token: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd

permissions:
  - user: admin@infrahq.com
    role: admin
  - user: jeff@infrahq.com
    role: edit
  - user: michael@infrahq.com
    role: view
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

### `users`

### Example

```yaml
permissions:
  - user: admin@infrahq.com
    role: admin
  - user: jeff@infrahq.com
    role: edit
  - user: michael@infrahq.com
    role: view
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

