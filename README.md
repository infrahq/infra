<br/>
<br/>
<br/>
<p align="center">
  <img src="./docs/images/logo.svg" height="48" />
</p>
<br/>

* Website: https://infrahq.com
* Docs: https://infrahq.com/docs
* Slack: https://infra-slack.slack.com

## Introduction
Identity and access management for Kubernetes. Instead of creating separate credentials and writing scripts to map permissions to Kubernetes, developers & IT teams can integrate existing identity providers (Okta, Google accounts, GitHub auth, Azure active directory) to securely provide developers with access to Kubernetes.

### Features
* One-command access: `infra login`
* Fine-grained permissions
* Onboard & offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)
* CLI & REST API
* Configure via `infra.yaml`

## Architecture

<p align="center">
  <br/>
  <br/>
  <img src="https://user-images.githubusercontent.com/251292/113448649-395cec00-93ca-11eb-9c70-ea4c5c9f82da.png" />
  <br/>
  <br/>
</p>


## Install

### Install Infra Engine on Kubernetes

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/early-access/master/deploy/kubernetes.yaml
```

Infra exposes a LoadBalancer endpoint:

```
$ kubectl get svc --namespace infra
NAME      TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra     LoadBalancer   10.12.11.116   31.58.101.169   80:32326/TCP   1m
```

Optionally, map a domain to the exposed endpoint (e.g. `infra.acme.com` to `31.58.101.169`).

### Install Infra CLI

Next, Install the Infra CLI:

```bash
# macOS
$ curl -L "https://github.com/infrahq/early-access/releases/latest/download/infra-darwin-$(uname -m)" -o /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Linux
$ curl -L "https://github.com/infrahq/early-access/releases/latest/download/infra-linux-$(uname -m)" -o /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Windows 10
$ curl.exe -L "https://github.com/infrahq/early-access/releases/download/latest/infra-windows-amd64.exe" -o infra.exe
```

## Usage

Configure Infra via `infra.yaml` with a single admin user:

```yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: infra
  namespace: infra
data:
  infra.yaml: |
    permissions:
      - user: example@acme.com
        permission: view
EOF
```

Create the user and login token via `kubectl`:

```
$ kubectl -n infra exec infra-0 -- infra user create example@acme.com
usr_js08jsec08

$ kubectl -n infra exec infra-0 -- infra token create usr_js08jsec08
sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7
```

Finally, log in as `example@acme.com` with the token created in the previous step:

```
$ infra login --token sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7 infra.acme.com
✔ Logging in with Token... success
✔ Logged in as example@acme.com
✔ Kubeconfig updated
```

That's it. You now have cluster access as `example@acme.com` with read-only `view` permissions.

```
$ kubectl get pods -A
kube-system   coredns-56b458df85-7z4ds          1/1     Running   0          2d4h
kube-system   coredns-56b458df85-wx48l          1/1     Running   0          2d4h
kube-system   kube-proxy-cxn9c                  1/1     Running   0          2d4h
kube-system   kube-proxy-nmnpb                  1/1     Running   0          2d4h
kube-system   metrics-server-5fbdc54f8c-nf85v   1/1     Running   0          46h

$ kubectl delete -n kube-system pod/kube-proxy-cxn9c # permission denied
```

## Documentation
* [Okta](./docs/okta.md)
* [Configuration Reference](./docs/configuration.md)
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
