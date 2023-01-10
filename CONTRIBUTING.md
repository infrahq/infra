# Contributing

Thank you so much for wanting to help make Infra successful. Community contributions are really important and help
us make Infra better.

## Types of Contributions

### Report Bugs or Suggest Enhancements

We use [GitHub Issues](https://github.com/infrahq/infra/issues) to track bug reports and feature requests. We're always
looking for ways to improve the project, and well written issues help us find things we may have missed. Before filing an issue though, please check to see if it has been filed before.

When filing the issue, we ask that you use good bug/feature etiquette. Make sure that you:

- Use a clear and descriptive title
- Include a description of what you expected to happen
- Attach a screenshot if relevant
- Include the Infra version you're using

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

1. Install [Go 1.19](https://go.dev/dl/#go1.19)
2. Install Postgres locally. On macOS:
   ```
   brew install postgresql
   brew services start postgresql
   ```
3. For building the infra Dashboard (under `ui/`), install Node. On macOS:
   ```
   brew install node
   ```

### Run the CLI locally

```
go run .
```

### Run the server

Run the server

```
export INFRA_DB_USERNAME=$(whoami)
go run . server -f ./dev/server.yaml
```

### Run the Dashboard

```
cd ui
npm install
npm run dev
```

### Sign up

Visit http://localhost/signup and create an organization (e.g. `acme`).

### Run the connector

Note: make sure your current Kubernetes context is set to the desired cluster.

Create a connector access key:

```
export INFRA_ACCESS_KEY=$(infra keys add --connector -q)
```

Then run the connector:

```
go run . connector -f ./dev/connector.yaml
```

### Run tests

```shell
go test ./...
```

Run tests with database:

```
POSTGRESQL_CONNECTION="host=localhost port=5432 user=$(whoami) dbname=postgres" go test ./...
```

#### Dashboard tests

```
cd ui
npm run test
```

### Linting

```shell
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
make lint
```

## Contributor License Agreement

All contributors also need to fill out the Infra's Contributor License Agreement (CLA) before changes can be merged. Infra's CLA assistant bot will automatically prompt for signatures from contributors on pull requests that require it.
