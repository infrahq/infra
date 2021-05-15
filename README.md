<br/>
<br/>
<br/>
<p align="center">
  <img src="./docs/images/logo.svg" height="48" />
</p>
<br/>
<br/>

## Introduction
Infra makes managing Kubernetes access easy & secure.

No more out-of-sync Kubeconfigs, lengthy scripts to map permissions, or untraceable service accounts. Instead, Infra enables teams to **dynamically** grant access to _right users or machines_ with the _right permissions_ for _the right amount of time_. Under the hood, Infra takes care of provisioning identities, creating & revoking credentials and generating fine-grained permissions on-the-fly.

<br/>
<p align="center">
  <img src="./docs/images/pic.svg" />
</p>
<br/>

### Features
* One-command access: `infra login`
* Fine-grained permissions
* Onboard & offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)
* CLI & REST API
* `infra.yaml` configuration file

## Documentation
* [Quickstart](./docs/quickstart.md)
* [Okta](./docs/okta.md)
* [CLI Reference](./docs/cli.md)
* [Configuration Reference](./docs/configuration.md)
* [API Reference](./docs/api.md)

## Develop

Clone the project:

```bash
git clone https://github.com/infrahq/infra
cd infra
```

Run locally:

```bash
go run .
```

## Test

Run tests:

```bash
go test ./...
```

## Release

Setup

* [GitHub CLI](https://github.com/cli/cli)
* [gon](https://github.com/mitchellh/gon) for signing MacOS binaries: `go get https://github.com/mitchellh/gon`

```
make release         # Build, sign and upload binaries
make release/docker  # Build and push Docker images
```

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
