<br/>
<br/>
<p align="center">
  <img src="./docs/images/logo.svg" height="48" />
  <br/>
  <br/>
</p>

## Introduction
Securely connect any user or machine to your Kubernetes clusters.

Infra is identity management built ground-up for Kubernetes. It integrates with existing identity providers like Okta or GitHub and makes it easy and secore for both humans & machines to access clusters with the right permissions. No more insecure all-or-nothing access, credential sharing, long scripts to map permissions, or identity provider sprawl.

### Features
* Secure access in one command: `infra login`
* Automatic credential rotation
* Sync users & groups via Okta _(Azure AD, GitHub, Google Accounts coming soon)_
* Fine-grained permissions
* CLI & REST API for programmatic access
* Audit logs for who did what, when _(coming soon)_

<br/>
<br/>
<p align="center">
  <img src="./docs/images/pic.svg" />
</p>
<br/>
<br/>


## Quickstart (Okta)

### Install Infra

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/deploy/kubernetes.yaml
```

Find which endpoint Infra is exposed on:

```
$ kubectl get svc --namespace infra
NAME      TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra     LoadBalancer   10.12.11.116   31.58.101.169   80:32326/TCP   1m
```

### Configure Okta

1. Create a  For **Application Name** choose **Infra**. For **Login redirect URIs** choose `http://localhost:2378/v1/redirect`
2. On the **General** tab, under **General Settings**, check the **Refresh Token** box.
3. Under the **Assignments** tab, assign Infra to one or more users or groups.
4. Next, click on the **Okta API Scopes** tab. Grant permissions for `okta.users.read` and `okta.groups.read`.
5. Back to the **General** tab, note the **Client ID** and **Client Secret**

### Configure Infra

Update Infra's secrets with your newly created Okta **Client Secret** from the previous step:

```
$ kubectl -n infra edit secret infra --from-literal=okta-client-secret=<client secret>
```

Next, update the Infra configuration:

```yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: infra
  namespace: infra
data:
  config.yaml: |
    providers:
      - okta:
          domain: acme.okta.com             # Your Okta domain
          client_id: 0oapn0qwiQPiMIyR35d6   # Client ID from previous step
    permissions:
      - provider: okta
        group: Developers
        permission: edit
      - provider: okta
        user: admin@acme.com          
        permission: admin
EOF
```

### Log in

Install the Infra CLI:

```bash
# macOS
$ brew install infrahq/tap/infra

# Windows
$ winget install --id com.infrahq.infra

# Linux
$ curl -L "https://github.com/infrahq/infra/releases/download/latest/infra-linux-$(uname -m)" -o /usr/local/bin/infra
```

Next, log in via the external IP exposed by Infra:

```
$ infra login 31.58.101.169
✔ Opening Okta login window
✔ Okta login successful
✔ Kubeconfig updated
✔ Context changed to `infra:jeff@acme.com`
```

Infra will automatically switch you to the new context. Now run any `kubectl` command as usual.

## Using Infra CLI

### Open an admin shell

```
kubectl -n infra exec -it infra-0 -- sh

$ infra users ls
ID                      NAME                  PROVIDER           CREATED                PERMISSION
usr_180jhsxjnxui1       jeff@acme.com         okta               2 minutes ago          admin
usr_mgna7u291s012       michael@acme.com      okta               2 minutes ago          view
```

### List users

List users that have been added to Infra:

```bash
$ infra users list
ID                      NAME                  PROVIDER           CREATED                PERMISSION
usr_180jhsxjnxui1       jeff@acme.com         okta               2 minutes ago          admin
usr_mgna7u291s012       michael@acme.com      okta               2 minutes ago          view
```

### Add a one-off user

```
$ infra users add bot@acme.com --permission=view
usr_oafia0301us10

To log in as bot@acme.com run:

infra login --token sk_Kc1dtcFazlIVFhkT2FsRjNaMmRGYVUxQk1kd18jdj10 31.58.101.169
```

### Delete a one-off user

```bash
$ infra users delete usr_mgna7u291s012
usr_mgna7u291s012
```

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
make release/docker  # Build and upload Docker image
```

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
