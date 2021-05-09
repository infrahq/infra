<br />
<br />
<img alt="Infra" src="https://user-images.githubusercontent.com/251292/117556309-5920a900-b035-11eb-9725-da10418b5333.png" height="60" />
<br />
<br />
<br />

## Introduction
Infra is Kubernetes Identity & Access Management (IAM) made easy. Securely connect any user or machine to any cluster.
<br/>
<br/>
<br/>
<br/>
![Architecture](https://user-images.githubusercontent.com/251292/117556405-a8b3a480-b036-11eb-9219-c28891e68e81.png)
<br/>
<br/>
<br/>

## Major features:
* One-command login
* Automatic Kubeconfigs & credential rotation
* Sync & log-in users via popular identity providers (Okta, Azure AD, GitHub, Google Accounts)
* Fine-grained permission templates for common tasks
* CLI & REST API for programmatic access
* _Coming soon:_ Audit logs (who did what, when)

## Quickstart (Okta)

### Deploy Infra via `kubectl`

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/deploy/infra.yaml
```

Wait for Kubernetes to expose an endpoint:

```
$ kubectl get svc --namespace infra
NAME      TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra     LoadBalancer   10.12.11.116   31.58.101.169   80:32326/TCP   1m
```

### Configure Infra

```yaml
$ kubectl -n infra apply -f - <
providers:
  - name: okta
    okta:
      domain: acme.okta.com   # Your okta domain
      users:                  # Okta users you'd like to sync
        - jeff@acme.com 
        - michael@acme.com

permissions:
  - user: jeff@acme.com
    permissions:              # permission templates: view, logs, exec, edit, admin
      - view
      - logs
    namespaces:               # optional namespaces to scope access
      - default
  - user: michael@acme.com
    permissions:
      - admin
```

### Log in as user

First install the Infra CLI:

```bash
# macOS
brew install infrahq/tap/infra

# Windows
winget install --id com.infrahq.infra

# Linux
curl -L "https://github.com/infrahq/infra/releases/download/latest/infra-linux-$(uname -m)" -o /usr/local/bin/infra
```

Then log in:

```
$ infra login 31.58.101.169
✔ Opening Okta login window
✔ Kubeconfig updated and context changed to `infra:jeff@acme.com`
```

That's it! Infra will automatically switch you to the new context. Now run any `kubectl` command as usual.

### Enabling Okta user sync

1. Create an Okta API key (see here)
2. Next, add your Okta API key to the secret created with Infra:

```
$ kubectl -n infra edit secret okta-api-key from-literal="okta-api-key=aj9d8023jad928dja928dja928"
```

## Using Infra CLI

### Open an admin shell

```
$ kubectl -n infra exec -it infra-0 -- sh
$ infra users ls
ID                      NAME                  PROVIDER           CREATED                PERMISSION
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
$ infra users add michael@acme.com
usr_mgna7u291s012

Please share the following login with michael@acme.com:

infra login --token sk_Kc1dtcFazlIVFhkT2FsRjNaMmRGYVUxQk1kd18jdj10 31.58.101.169
```

### Delete a one-off user

```bash
$ infra users delete michael@acme.com
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
