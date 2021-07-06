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

## Release

```bash
# Build and publish binaries
make release

# Build and publish helm charts
make release/helm

# Build and publish Docker images
make release/docker
```
