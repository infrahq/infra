<br/>
<br/>
<p align="center">
  <img src="./docs/images/logo.svg" height="48" />
  <br/>
  <br/>
</p>

## Introduction
Securely connect any user or machine to your Kubernetes clusters.

Infra is identity management built ground-up for Kubernetes. It provides an easy, secure, streamlined way for both humans & machines to access clusters. No more insecure all-or-nothing access, credential sharing, long scripts to map permissions, or identity provider sprawl.

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

1. Create a [new Okta OIDC App](https://dev-02708987-admin.okta.com/admin/apps/oauth2-wizard/create?applicationType=WEB). For **Application Name** choose **Infra**. For **Login redirect URIs** choose `http://localhost:3001`
2. Next, click on the **Okta API Scopes** tab. Grant permissions for `okta.users.read` and `okta.groups.read`.
3. Go back to the **General** tab. Take note of the **Client ID** and **Client secret** for the next step.

### Configure Infra

First, update Infra's secrets with your newly created Okta **Client ID** and **Client secret** from the last step:

```bash
$ kubectl -n infra edit secret infra --from-literal=okta-client-id=<your okta client id> --from-literal=okta-client-secret=<your okta client secret>
```

Next, update the Infra configuration

```yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: infra
  namespace: infra
data:
  config.yaml: |
    okta:
      domain: acme.okta.com       # Your okta domain
      groups:
        - developers              # Groups to sync
      users:
        - admin@acme.com          # Individual users to sync
    permissions:
      - group: developers
        permissions:              # permission templates: view, logs, exec, edit, admin
          - edit
      - user: admin@acme.com
        permissions:
          - admin
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

### Inspect a user's permissions

```bash
$ infra user permissions jeff@acme.com
KUBERNETES RESOURCE                                           LIST  CREATE  UPDATE  DELETE
daemonsets.apps                                               ✔     ✔       ✔       ✔
daemonsets.extensions                                         ✔     ✔       ✔       ✔
deployments.apps                                              ✔     ✔       ✔       ✔
deployments.extensions                                        ✔     ✔       ✔       ✔
endpoints                                                     ✔     ✔       ✔       ✔
events                                                        ✔     ✔       ✔       ✔
events.events.k8s.io                                          ✔     ✔       ✔       ✔
pods                                                          ✔     ✔       ✔       ✔
pods.metrics.k8s.io                                           ✔                     
podsecuritypolicies.extensions                                ✔     ✔       ✔       ✔
podsecuritypolicies.policy                                    ✔     ✔       ✔       ✔
replicasets.apps                                              ✔     ✔       ✔       ✔
replicasets.extensions                                        ✔     ✔       ✔       ✔
replicationcontrollers                                        ✔     ✔       ✔       ✔
resourcequotas                                                ✔     ✔       ✔       ✔
rolebindings.rbac.authorization.k8s.io                        ✔     ✔       ✔       ✔
roles.rbac.authorization.k8s.io                               ✔     ✔       ✔       ✔
runtimeclasses.node.k8s.io                                    ✔     ✔       ✔       ✔
secrets                                                       ✔     ✔       ✔       ✔ 
selfsubjectaccessreviews.authorization.k8s.io                       ✔               
selfsubjectrulesreviews.authorization.k8s.io                        ✔               
serviceaccounts                                               ✔     ✔       ✔       ✔
services                                                      ✔     ✔       ✔       ✔
statefulsets.apps                                             ✔     ✔       ✔       ✔
storageclasses.storage.k8s.io                                 ✔     ✔       ✔       ✔
subjectaccessreviews.authorization.k8s.io                           ✔               
tokenreviews.authentication.k8s.io                                  ✔               
validatingwebhookconfigurations.admissionregistration.k8s.io  ✔     ✔       ✔       ✔
volumeattachments.storage.k8s.io                              ✔     ✔       ✔       ✔
```

### Add a one-off user

```bash
$ infra users add michael@acme.com --permission=view
User added. Please share the following login with michael@acme.com:

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
