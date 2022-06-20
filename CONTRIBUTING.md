# Contributing

Thank you so much for wanting to help make Infra successful. Community contributions are really important and help
us make Infra better.

## Types of Contributions

### Report Bugs or Suggest Enhancements

We use [GitHub Issues](https://github.com/infrahq/infra/issues) to track bug reports and feature requests. We're always
looking for ways to improve the project, and well written issues help us find things we may have missed. Before filing an issue though,
please check to see if it has been filed before.

When filing the issue, we ask that you use good bug/feature etiquette. Make sure that you:
 * Use a clear and descriptive title
 * Include a description of what you expected to happen
 * Attach a screenshot if relevant
 * Include the Infra and Kubernetes versions you're using
 * Describe where you're running Kubernetes


### Fix a Bug or Implement a Feature

If you'd like to help fix any bugs or contribute a new feature, please first make sure there is an [issue](https://github.com/infrahq/infra/issues) filed.

Any issues tagged with `bug` or `good first issue` make great places to start. Issues tagged with `enhancement` are
changes that we're thinking of making in the future. If you want to talk about an issue more, check out our [discussions page](https://github.com/infrahq/infra/discussions).

Our repository follows GitHub's normal forking model. If you have never forked a repository before, follow GitHub's
documentation which describes how to:
  * [Fork a repo](https://docs.github.com/en/get-started/quickstart/fork-a-repo)
  * [Contribute to projects](https://docs.github.com/en/get-started/quickstart/contributing-to-projects)

When you are ready to commit your change, follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
for your commit message. The type must be one of `fix`, `feat`, `improve`, or `maintain`. These types are
documented in the [commitlint config file](.github/commitlint.config.js).

## Developing Infra

### Setup

1. Install [Go 1.18](https://go.dev/dl/#go1.18)
1. Clone the project
    ```
    git clone https://github.com/infrahq/infra
    cd infra
    ```

### Run locally

```
go run .
```

### Run tests

```
go test ./...

# for shorter tests
go test -short ./...
```

### Linting

```
make tools
make lint
```

### Run in Docker Desktop

#### Prerequisites

Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) and enable Kubernetes.

#### Build and run

Run `make dev`:

```
make dev
```

#### Customize

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

## Contributor License Agreement

All contributors also need to fill out the Infra's Contributor License Agreement (CLA) before changes can be merged. Infra's CLA assistant bot will automatically prompt for signatures from contributors on pull requests that require it.

