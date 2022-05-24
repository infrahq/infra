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

## Run tests

```bash
go test ./...

# for shorter tests
go test -short ./...
```

## Linting

```bash
make tools
make lint
```

## Run in Docker Desktop

### Prerequisites

Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) and enable Kubernetes.

### Build and run

Run `make dev`:

```bash
make dev
```

### Customize

The local Kubernetes setup uses [`helm`](https://helm.sh/) can be modified via a `values.yaml` file:

```bash
make dev flags="-f values.yaml"
```

Example `values.yaml` files:

* Enable the in-cluster connector

```yaml
---
# example values.yaml
server:
  config:
    users:
      - name: admin@local.dev
        password: password

    grants:
      - user: admin@local.dev
        role: admin
        resource: infra

connector:
  enabled: true
  config:
    name: desktop
```

> Note: login via `infra login` with the username `admin@local.dev` and password `password`


* Disable volumes and persistence

```yaml
# example values.yaml
---
server:
  persistence:
    enabled: false
```

See [Helm Chart reference](./reference/helm-chart.md) for a complete list of options configurable through Helm.

