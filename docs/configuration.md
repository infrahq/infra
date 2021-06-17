# Configuration Reference

* [Example](#example)
* [ConfigMap Usage](#configmap-usage)
* [Reference](#reference)
  * [`sources`](#sources)
    * [`okta`](#okta)
  * [`permissions`](#permissions)
    * [`user`](#user)
    * [`destination`](#destination)
    * [`role`](#role)

## Overview

For teams who require configuration to be stored in version control, Infra can be managed via a configuration file, `infra.yaml`.

**Important:** when configuring keys such as `sources` or `permissions` via `infra.yaml`, any manual edits to this data will be overwritten when Infra Server restarts.

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
    sources:
      - kind: okta
        domain: example.okta.com
        clientId: 0oapn0qwiQPiMIyR35d6
        clientSecret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
        apiToken: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd

    permissions:
      - user: admin@example.com
        destination: production
        role: admin
      - user: michael@example.com
        destination: production
        role: view
EOF
```

Then, restart Infra server to apply the change:

```
kubectl rollout restart -n infra deployment/infra
```

## Full Example

```yaml
sources:
  - kind: okta
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
    api-token: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd

permissions:
  - user: admin@example.com
    destination: production
    role: kubernetes.admin
```


## Reference

### `sources`

#### `okta`

* `domain`: Okta domain
* `client-id`: Client ID for the Okta application
* `client-secret`: Client Secret for the Okta application
* `api-token`: Okta API Token

Example:

```yaml
sources:
  - kind: okta
    domain: acme.okta.com
    client-id: 0oapn0qwiQPiMIyR35d6
    client-secret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
    api-token: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd
```

### `permissions`

### Example

```yaml
permissions:
  - user: admin@infrahq.com
    destination: production
    role: kubernetes.admin
  - user: jeff@infrahq.com
    destination: production
    role: kubernetes.viewer
  - user: michael@infrahq.com
    destination: production
    role: kubernetes.editor
```

### `user`

`user` is a user's email

### `destination`

`destination` is a target destination to grant access to, e.g. the kubernetes cluster name

### `role`

`role` defines a permission level, giving users access to specific resources they need

| Role                    | Description                        |
| :--------               | :------------------------------    |
| kubernetes.viewer       | Read-only for most resources       |
| kubernetes.editor       | Read & write most resources        |
| kubernetes.admin        | Read & write any resource          |
