# Developing Infra

## Setup

1. Install [Go 1.18](https://go.dev/dl/#go1.18rc1)
1. Clone the project

    ```bash
    git clone https://github.com/infrahq/infra
    cd infra
    ```

1. Install tools

    ```bash
    go get
    make tools
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

* Disabling persistence

```yaml
# example infra.yaml
---
server:
  persistence:
    enabled: false

engine:
  pesistence:
    enabled: false
```

* Disabling telemetry and crash reporting

```yaml
# example infra.yaml
---
server:
  config:
    enable-telemetry: false
    enable-crash-reporting: false
```

* Enabling Kubernetes LoadBalancer

```yaml
# example infra.yaml
---
server:
  service:
    type: LoadBalancer

engine:
  service:
    type: LoadBalancer
    securePort: 8443  # to avoid colliding with infra-server's securePort (443)
```

See [Helm Chart reference](./helm.md) for a complete list of options configurable through Helm.

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
