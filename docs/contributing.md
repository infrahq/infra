# Developing Infra

## Setup

* Install [Go](https://golang.org/doc/install) or via `brew install go`
* Install [Docker Desktop](https://www.docker.com/products/docker-desktop) or if on Linux, [Docker Engine](https://docs.docker.com/engine/install/).
* Install [`openapi-generator`](https://openapi-generator.tech/docs/installation/) via `brew install openapi-generator`

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
