<br/>
<br/>
<p align="center">
  <img src="./docs/images/logo.svg" height="48" />
  <br/>
  <br/>
</p>

## Introduction
Infra makes managing Kubernetes access easy & secure.

No more out-of-sync Kubeconfigs, lengthy scripts to map permissions, or untraceable service accounts.

Instead, Infra enables teams to **dynamically** grant access to _right users or machines_ with the _right permissions_ for _the right amount of time_. Under the hood, Infra takes care of provisioning identities, creating & revoking credentials and generating fine-grained permissions on-the-fly.

<br/>
<p align="center">
  <img src="./docs/images/pic.svg" />
</p>
<br/>

### Features
* Cluster access in one command: `infra login`
* Fine-grained permissions
* Onboard & offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)
* CLI & REST API for programmatic access
* `infra.yaml` configuration file

## Demo

```
$ infra login infra.acme.com
[x] Okta
[ ] GitHub
[ ] Google Accounts
[ ] Token

✔ Logging in with Okta... success
✔ Logged in as michael@acme.com
✔ Kubeconfig updated

$ kubectl get pods -A
kube-system           coredns-55ff57f948-h6jjk                         1/1     Running   0          6d7h
kube-system           coredns-55ff57f948-w8prx                         1/1     Running   0          6d7h
kube-system           kube-proxy-5flfd                                 1/1     Running   0          6d7h
kube-system           kube-proxy-f952f                                 1/1     Running   0          6d7h
kube-system           kube-proxy-t99vf                                 1/1     Running   0          6d7h
kube-system           metrics-server-5fbdc54f8c-5m84f                  1/1     Running   0          6d5h
...
```

## Install

### Install Infra Engine

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/deploy/kubernetes.yaml
```

Find the endpoint on which Infra Engine is exposed:

```
$ kubectl get svc --namespace infra
NAME      TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra     LoadBalancer   10.12.11.116   31.58.101.169   80:32326/TCP   1m
```

In this case Infra Engine will be exposed on `31.58.101.169`.

### Install Infra CLI

```bash
# macOS
$ curl --url "https://github.com/infrahq/infra/releases/download/latest/infra-darwin-$(uname -m)" --output /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Linux
$ curl --url "https://github.com/infrahq/infra/releases/download/latest/infra-linux-$(uname -m)" --output /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Windows 10
$ curl.exe --url "https://github.com/infrahq/infra/releases/download/latest/infra-windows-amd64.exe" --output infra.exe
```

## Documentation
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
