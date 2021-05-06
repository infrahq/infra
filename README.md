<p align="center">
  <br/>
  <br/>
  <img src="https://user-images.githubusercontent.com/3325447/109728544-68423100-7b84-11eb-8fc0-759df7c3b974.png" height="128" />
  <br/>
  <br/>
</p>

* Website: https://infrahq.com
* Docs: https://infrahq.com/docs
* Slack: https://infra-slack.slack.com

## Introduction

Identity and access management for Kubernetes. Instead of creating separate credentials and writing scripts to map permissions to Kubernetes, developers & IT teams can integrate existing identity providers (Okta, Google accounts, GitHub auth, Azure active directory) to securely provide developers with access to Kubernetes.

## Use cases
- Fine-grained permissions
- Mapping existing users & groups (in Okta, Azure AD, Google, etc) into Kubernetes groups
- On-boarding and off-boarding users (automatically sync users against identity providers)
- No more out of sync Kubeconfig
- Cloud vendor-agnostic
- Coming soon: Audit logs (who did what, when)

## Architecture

<p align="center">
  <br/>
  <br/>
  <img src="https://user-images.githubusercontent.com/251292/113448649-395cec00-93ca-11eb-9c70-ea4c5c9f82da.png" />
  <br/>
  <br/>
</p>

## Installing on Kubernetes

Deploy via `kubectl`:

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/deploy/infra.yaml
```

Wait for Kubernetes to expose an endpoint:

```
$ kubectl get svc --namespace infra
NAME      TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra     LoadBalancer   10.12.11.116   31.58.101.169   80:32326/TCP   1m
```

Finally, create your first user:

```
$ kubectl exec -it --namespace infra infra-0 -- infra users add jeff@acme.com

User jeff@acme.com added. Please share the following command with them so they can log in:

infra login --token sk_EFI4dPZQjEnPTYG5JCL4mr0mOQDuloTVyR1HjlpPlEaITQZM 31.58.101.169
```

## Installing Infra CLI

```
# macOS
brew install infrahq/tap/infra

# Windows
winget install --id com.infrahq.infra

# Linux
curl -L "https://github.com/infrahq/infra/releases/download/latest/infra-linux-$(uname -m)" -o /usr/local/bin/infra
```

## Usage Examples

### Log in

To log in as your first user, run the `infra login` command generated above:

```
$ infra login --token sk_EFI4dPZQjEnPTYG5JCL4mr0mOQDuloTVyR1HjlpPlEaITQZM 31.58.101.169
Kubeconfig updated.
```

### List users

List users that have been added to Infra:

```
$ infra users list
ID                      NAME                  PROVIDER           CREATED           
usr_180jhsxjnxui1       jeff@acme.com         infra              2 minutes ago
```

### List users with permissions

List users that have been added to Infra:

```
$ infra users list -a
ID                      NAME                  PROVIDER           CREATED             NAMESPACE          ROLE
usr_180jhsxjnxui1       jeff@acme.com         infra              2 minutes ago       *                  view
                                                                                     wordpress          edit
usr_xm97sqlhgau40       michael@acme.com      infra              5 minutes ago       *                  view
                                                                                     wordpress          edit
```

### Inspect a user

```
$ infra users inspect jeff@acme.com --namespace wordpress
NAME                                                          LIST  CREATE  UPDATE  DELETE
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

### Add a user

```
$ infra users add michael@acme.com

Please share the following login with michael@acme.com:

infra login --token sk_Kc1dtcFazlIVFhkT2FsRjNaMmRGYVUxQk1kd18jdj10 31.58.101.169
```

### Delete a user

```
$ infra users delete michael@acme.com
User deleted.
```

### Connect a provider

```
$ infra providers add okta
✔ Domain: acme.okta.com

Create a scoped API key by following instructions: https://infrahq.com/docs/okta
✔ API Key: "SWSS ajd80aj2071h0h0e7fh20h3f03gf02g6q3fg293o6fg2369"

✔ Okta added.
```

### List groups and users from a provider

```
$ infra groups list
ID                      NAME                PROVIDER        USERS          CREATED
grp_ka93j10j48wl9       admin               okta            2              2 minutes ago
grp_smd810sk18720       developers          okta            6              2 minutes ago

$ infra users list
ID                      NAME                  PROVIDER           CREATED
usr_xm97sqlhgau40       michael@acme.com      infra okta         10 minutes ago
usr_180jhsxjnxui1       jeff@acme.com         infra okta         13 minutes ago
usr_aja2od8a2od8a       stu@acme.com          okta               2 minutes ago
usr_nv92379237ahl       suzie@acme.com        okta               2 minutes ago
usr_xm97sqlhgau40       lucy@acme.com         okta               2 minutes ago
usr_cm0a8jf38a021       joe@acme.com          okta               2 minutes ago
usr_oz9783197911b       brian@acme.com        okta               2 minutes ago
usr_4hv6s9ah27dsj       pete@acme.com         okta               2 minutes ago
```

## CLI Reference

```
$ infra
Infra Engine

Usage:
  infra [command]
  infra [flags]

Available Commands:
  users         Manage users
  groups        Manage groups
  providers     Manage identity providers
  login         Log in to an Infra Engine
  start         Start Infra Engine

Flags:
  -h, --help    Print more information about a command

Use "infra [command] --help" for more information about a command.
```

## Configuration

For scriptability, Infra Engine can be configured using a yaml file

```yaml
providers:
  - name: acme-okta
    kind: okta
    config:
      api_key: /etc/infra/okta-api-key
      domain: acme.okta.com
    groups:
      - developers
      - admins

permissions:
  - provider: acme-okta
    group: developers
    role: view
  - provider: acme-okta
    group: admins
    clusterRole: admin
```

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
