# Developing Infra

## Setup

1. Install [Go 1.18](https://go.dev/dl/#go1.18)
1. Clone the project

    ```bash
    git clone https://github.com/infrahq/infra
    cd infra
    ```

## Run locally

```bash
go run .
```

## Run a full local setup

### Setup

* Install [Docker](https://docker.com/)
  * (macOS, Windows) [Docker Desktop](https://www.docker.com/products/docker-desktop)
  * (Linux) [Docker Engine](https://docs.docker.com/engine/install)
* `envsubst`
  * (macOS, Linux) `brew install gettext`

---

The local setup can be customized with environment varibles. Some are required for a functional deployment.

| Name             | Description                                                   | Default               |
|------------------|---------------------------------------------------------------|-----------------------|
| `NAMESPACE`      | Kubernetes namespace to install `infra`                       | `""`                  |
| `IMAGE_TAG`      | Docker tag                                                    | `0.0.0-development`   |
| `VALUES`         | Values file to pass to Helm                                   | `docker-desktop.yaml` |

```bash
make dev
```

### Customizing local setup beyond the basics

If further customization is required, additional values files can be supplied to the installation by modifying the `VALUES` environment variable. It is recommended to append to `VALUES` rather than fully overriding it.

Some common configurations for local development include:

* Disable persistence

```yaml
# example infra.yaml
---
server:
  persistence:
    enabled: false
```

* Disable telemetry

```yaml
# example infra.yaml
---
server:
  config:
    enable-telemetry: false
```

* Enable in-cluster connector

> Note: enabling in-cluster connector disables first-time user signup and requires an admin user to be created by Helm

```yaml
---
# example infra.yaml
server:
  additionalIdentities:
    - name: admin
      password: PASSWORD

  additionalGrants:
    - user: admin
      role: admin
      resource: infra

connector:
  enabled: true
```

See [Helm Chart reference](./reference/helm-chart.md) for a complete list of options configurable through Helm.

```bash
export VALUES='infra.yaml docker-desktop.yaml'
make dev
```

Or

```bash
VALUES='infra.yaml docker-desktop.yaml' make dev
```

Or

```bash
make dev VALUES='infra.yaml docker-desktop.yaml'
```
