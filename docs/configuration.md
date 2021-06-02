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
  okta:
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
    api-token: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd

permissions:
  - name: admin
    users: ["admin@example.com"]
  - name: write
    users: ["suzie@example.com", "john@example.com"]
  - name: readonly
    users: ["bob@example.com", "tony@example.com", "alice@example.com"]
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
  okta:
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
    api-token: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd
```

### `users`

### Example

```yaml
permissions:
  - name: admin
    users: ["admin@example.com"]
  - name: write
    users: ["suzie@example.com", "john@example.com"]
  - name: readonly
    users: ["bob@example.com", "tony@example.com", "alice@example.com"]
```

### `user`

`user` is a user's email

### `permission`

`permission` defines a permission level, giving users access to specific resources and tasks they need

| Permission | Description                        |
| :--------  | :------------------------------    |
| view       | Read-only for most resources       |
| edit       | Read & write most resources        |
| admin      | Read & write any resource          |

