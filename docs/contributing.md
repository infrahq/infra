# Developing Infra

## Setup

* Install [Go](https://golang.org/doc/install) or via `brew install go`
* Install [Docker Desktop](https://www.docker.com/products/docker-desktop) or if on Linux, [Docker Engine](https://docs.docker.com/engine/install/).

Clone the project:

```
git clone https://github.com/infrahq/infra
cd infra
```

Install tools:

```
go get
make tools
```

Run locally:

```
go run .
```

Run a full setup (Infra Registry + Infra Engine):

```
make dev
```

## Generate docs

```
make docs
```

## Test

Run tests:

```
make test
```

## Releasing a new version of Infra

### 1. Bump Helm versions

First, bump both the chart and app versions in `helm`:

* https://github.com/infrahq/infra/blob/main/helm/charts/infra/Chart.yaml
* https://github.com/infrahq/infra/blob/main/helm/charts/infra/charts/engine/Chart.yaml

Then, commit these changes and push them to `main` (or create a PR).

### 2. Create and push a tag

```
git tag v0.0.7
git push --tags
```

A new version of Infra will be prepared and released via GitHub actions.

### 3. Verify release

Verify the [release job](https://github.com/infrahq/infra/actions/workflows/release.yml) succeeded.

## Manual release

```bash
# Build and publish binaries
make release

# Build and publish helm charts
make release/helm

# Build and publish Docker images
make release/docker
```
