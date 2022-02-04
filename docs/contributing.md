# Developing Infra

## Setup

1. Install [Go 1.18](https://go.dev/dl/#go1.18beta1)
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

## Releasing a new version of Infra

### 1. Create a release on GitHub, marked as pre-release

1. Head over to https://github.com/infrahq/infra/releases and click **Draft a new Release**
2. Fill out the form with the new version as the tag and release name as shown below:

![release screenshot](https://user-images.githubusercontent.com/5853428/137145290-7edef0ce-658b-4b78-b76c-663490ce547a.png)


3. Click **Publish Release**

A new version of Infra will be prepared and released via GitHub actions.

### 2. Verify release

Verify the [release job](https://github.com/infrahq/infra/actions/workflows/release.yml) succeeded.

### 3. Mark as released

1. Navigate back to https://github.com/infrahq/infra/releases and click on your release.
2. Uncheck **This is a pre-release**

You're done!