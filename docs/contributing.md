# Developing Infra

## Setup

1. Install [Go 1.18](https://go.dev/dl/#go1.18beta1)
2. Clone the project

    ```bash
    git clone https://github.com/infrahq/infra
    cd infra
    ```

3. Install tools

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
| `OKTA_DOMAIN`    | Okta domain URL                                               | (required)            |
| `OKTA_CLIENT_ID` | Okta client ID                                                | (required)            |
| `OKTA_SECRET`    | Kubernetes secret containing Okta client secret and API token | (required)            |
| `NAMESPACE`      | Kubernetes namespace to install `infra`                       | `""`                  |
| `IMAGE_TAG`      | Docker tag                                                    | `0.0.0-development`   |
| `VALUES`         | Values file to pass to Helm                                   | `docker-desktop.yaml` |

```bash
make dev
```

### Customizing local setup beyond the basics

If further customization is required, additional values files can be supplied to the installation by modifying the `VALUES` environment variable. It is recommended to append to `VALUES` rather than fully overriding it.

```yaml
# example infra.yaml
---
enable-telemetry: false
enable-crash-reporting: false
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

## Generate OpenAPI Clients

### Setup

* Install [OpenAPI Generator](https://openapi-generator.tech/docs/installation)

---

```bash
make openapi
```

## Generate docs

```bash
make docs
```

## Test

Run tests:

```bash
make test
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

## Manual release

To release from your local computer, switch to the tag you want to release (e.g. `v0.0.8`) and run:

```bash
# Build and publish binaries
make release

# Build and publish helm charts
make release/helm

# Build and publish Docker images
make release/docker
```
