<br/>
<br/>
<br/>
<p align="center">
  <img src="./docs/images/logo.svg" height="48" />
</p>
<br/>
<br/>

## Introduction
Infra makes managing user access to Kubernetes easy & secure.

No more out-of-sync Kubeconfigs, lengthy scripts to map permissions, or untraceable service accounts. 

Infra enables teams to **dynamically** grant access to _right users or machines_ with the _right permissions_ for _the right amount of time_. Under the hood, Infra takes care of provisioning identities, creating & revoking credentials and generating fine-grained permissions on-the-fly.

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
* Configure via `infra.yaml`

### Quickstart

####  1. Deploy Infra Engine

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/deploy/kubernetes.yaml
...

$ kubectl get svc --namespace infra
NAME      TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra     LoadBalancer   10.12.11.116   31.58.101.169   80:32326/TCP   1m
```

Optionally, map a domain (e.g. `infra.acme.com` to `31.58.101.169`).

Next, generate an admin token.

```
$ kubectl exec -n infra infra-0 -- infra token create --user jeffadmin@acme.com
sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7
```

#### 2. Configure

Add users:

```yaml
$ cat <<EOF | kubectl -n infra apply -f -
users:
  - email: admin@acme.com
    permission: admin
  - email: jeff@acme.com
    permission: edit
    namespace: default
```

#### 3. Login

Install Infra CLI

```bash
# macOS
$ curl --url "https://github.com/infrahq/infra/releases/download/latest/infra-darwin-$(uname -m)" --output /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Linux
$ curl --url "https://github.com/infrahq/infra/releases/download/latest/infra-linux-$(uname -m)" --output /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Windows 10
$ curl.exe --url "https://github.com/infrahq/infra/releases/download/latest/infra-windows-amd64.exe" --output infra.exe
```

```
$ infra login --token sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7 infra.acme.com
✔ Logging in with Okta... success
✔ Logged in as admin@acme.com
✔ Kubeconfig updated
```

That's it. You now have cluster access as admin@acme.com.

## Documentation
* [Managing users](./docs/users.md)
* [Okta](./docs/okta.md)
* [Configuration File](./docs/configuration.md)
* [CLI Reference](./docs/cli.md)
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
