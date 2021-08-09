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

## Working on the UI

Developing the UI requires two tabs:

```
# In first terminal window, run next.js dev server
cd internal/registry/ui
npm install
npm run dev

# In second terminal window, run Go server
go run . registry --ui-proxy=http://localhost:3000
```

To build a static version of the ui that can be imported into the Go server:

```
make generate
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

![release screenshot](https://user-images.githubusercontent.com/251292/124816016-00d32e00-df36-11eb-9b99-95b304195c75.png)

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
