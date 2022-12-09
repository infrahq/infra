# Contributing

Thank you so much for wanting to help make Infra successful. Community contributions are really important and help
us make Infra better.

## Types of Contributions

### Report Bugs or Suggest Enhancements

We use [GitHub Issues](https://github.com/infrahq/infra/issues) to track bug reports and feature requests. We're always
looking for ways to improve the project, and well written issues help us find things we may have missed. Before filing an issue though,
please check to see if it has been filed before.

When filing the issue, we ask that you use good bug/feature etiquette. Make sure that you:

- Use a clear and descriptive title
- Include a description of what you expected to happen
- Attach a screenshot if relevant
- Include the Infra and Kubernetes versions you're using
- Describe where you're running Kubernetes

### Fix a Bug or Implement a Feature

If you'd like to help fix any bugs or contribute a new feature, please first make sure there is an [issue](https://github.com/infrahq/infra/issues) filed.

Any issues tagged with `bug` or `good first issue` make great places to start. Issues tagged with `enhancement` are
changes that we're thinking of making in the future. If you want to talk about an issue more, check out our [discussions page](https://github.com/infrahq/infra/discussions).

Our repository follows GitHub's normal forking model. If you have never forked a repository before, follow GitHub's
documentation which describes how to:

- [Fork a repo](https://docs.github.com/en/get-started/quickstart/fork-a-repo)
- [Contribute to projects](https://docs.github.com/en/get-started/quickstart/contributing-to-projects)

## Developing Infra

### Setup

1. Install [Go](https://go.dev/dl), version 1.19 or higher
1. Clone the project
   ```shell
   git clone https://github.com/infrahq/infra
   cd infra
   ```

### Run locally

```shell
go run .
```

### Run tests

```shell
go test ./...

# for shorter tests
go test -short ./...
```

### Linting

```shell
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
make lint
```

### Run in Kubernetes

#### Setup

1. Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) and enable Kubernetes.
1. Add the `infrahq` Helm repo

   ```shell
   helm repo add infrahq https://infrahq.github.io/helm-charts
   helm repo update
   ```

For a full list of configurable options, use `helm show values infrahq/infra` and `helm show values infrahq/infra-server`.

#### Install the server

```shell
make dev
```

#### Install the connector

```shell
make dev/connector flags='--set config.accessKey=... --set config.server.url=...'
```

###

### CLI documentation

To generate CLI documentation, run the `docgen` package:

```shell
go run ./internal/docgen
```

## Contributor License Agreement

All contributors also need to fill out the Infra's Contributor License Agreement (CLA) before changes can be merged. Infra's CLA assistant bot will automatically prompt for signatures from contributors on pull requests that require it.
