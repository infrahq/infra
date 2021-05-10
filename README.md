<br/>
<br/>
<p align="center">
  <img src="./docs/images/logo.svg" height="48" />
  <br/>
  <br/>
</p>

## Introduction
Securely connect any user or machine to your Kubernetes clusters.

Infra is identity management built ground-up for Kubernetes. It integrates with existing identity providers like Okta or GitHub and makes it easy and secure for both humans & machines to access clusters with the right permissions.

No more insecure all-or-nothing access, credential sharing, long scripts to map permissions, or identity provider sprawl.

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

### Configure Okta

1. Log into Okta as an Administrator
2. Under the left menu click **Applications > Applications**. Click **Add Application** then **Create New App**. Select "OpenID Connect" from the dropdown, then click **Create**
3. For **Application name** write **Infra**. Optionally: add [the infra logo](./docs/images/okta.png). For **Login redirect URIs** write `http://localhost:2379/auth/callback`. Click **Save**.
4. On the **General** tab, under **General Settings**, click **Edit** then enable the **Refresh Token** checkbox.
5. Under the **Assignments** tab, assign Infra to one or more users or groups.
6. Next, click on the **Okta API Scopes** tab. Grant permissions for `okta.users.read` and `okta.groups.read`.
7. Back to the **General** tab, note the **Client ID** and **Client Secret** for the next step.

### Install and configure Infra

Install Infra via `kubectl`:

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/deploy/kubernetes.yaml
```

Update Infra's secrets with your newly created Okta **Client Secret** from the previous step:

```
$ kubectl -n infra edit secret infra --from-literal=okta-client-secret=In6P_qEoEVugEgk_7Z-Vkl6CysG1QapBBCzS5O7m # Client Secret from previous step
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
          domain: acme.okta.com             # (Replace) Your Okta domain
          client-id: 0oapn0qwiQPiMIyR35d6   # (Replace) Client ID from previous step
    permissions:
      - user: *                             # Give all assigned users view permissions by default
        permission: view                    # Possible permissions: view, edit, admin
      - user: admin@acme.com                # (Replace/remove) Give a single user admin permission
        permission: admin
      - group: devops                       # (Replace/remove) Give a group edit permission. This is the name of your group in Okta
        permission: edit
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

Find the endpoint which Infra is exposed on:

```
$ kubectl get svc --namespace infra
NAME      TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra     LoadBalancer   10.12.11.116   31.58.101.169   80:32326/TCP   1m
```

In this case Infra is exposed on `31.58.101.169`

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
