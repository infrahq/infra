# Configuration Reference

* [Example](#example)
* [ConfigMap Usage](#configmap-usage)
* [Reference](#reference)
  * [`providers`](#providers)
    * [`okta`](#okta)
  * [`grants`](#grants)
    * [`user`](#user)
    * [`resource`](#resource)
    * [`role`](#role)

## Overview

For teams who require configuration to be stored in version control, Infra can be managed via a configuration file, `infra.yaml`.

**Important:** when configuring keys such as `providers` or `grants` via `infra.yaml`, any manual edits to this data will be overwritten when Infra Server restarts.

## Kubernetes ConfigMap Example

To specify via Kubernetes, create a ConfigMap as show below:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: infra
  namespace: infra
data:
  infra.yaml: |
    providers:
      - kind: okta
        domain: example.okta.com
        client-id: 0oapn0qwiQPiMIyR35d6
        client-secret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
        api-token: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd

    grants:
      - user: admin@example.com
        resource: production
        role: kubernetes.admin
      - user: michael@example.com
        resource: production
        role: kubernetes.viewer
      - user: admin@example.com
        resource: infra
        role: infra.owner
EOF
```

Then, restart Infra server to apply the change:

```
kubectl rollout restart -n infra deployment/infra
```

## Full Example

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
    role: kubernetes.admin
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
    resource: production
    role: kubernetes.admin
  - user: jeff@infrahq.com
    resource: production
    role: kubernetes.viewer
  - user: michael@infrahq.com
    resource: production
    role: kubernetes.editor
```

### `user`

`user` is a user's email

### `resource`

`resource` is a target resource to grant access to, e.g. the kubernetes cluster name

### `role`

`role` defines a permission level, giving users access to specific resources and tasks they need

| Role                    | Description                        |
| :--------               | :------------------------------    |
| kubernetes.viewer       | Read-only for most resources       |
| kubernetes.editor       | Read & write most resources        |
| kubernetes.admin        | Read & write any resource          |
