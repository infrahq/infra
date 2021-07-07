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

Then, commit these changes and push them to the `main` branch (or create a PR).

### 2. Create a release on GitHub, marked as pre-release

1. Head over to https://github.com/infrahq/infra/releases and click **Draft a new Release**
2. Fill out the form with the new version as the tag and release name as shown below:

![release screenshot](https://user-images.githubusercontent.com/251292/124816016-00d32e00-df36-11eb-9b99-95b304195c75.png)

3. Click **Publish Release**

A new version of Infra will be prepared and released via GitHub actions.

### 3. Verify release

Verify the [release job](https://github.com/infrahq/infra/actions/workflows/release.yml) succeeded.

### 4. Mark as released

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
