# Infra

![infra](https://user-images.githubusercontent.com/3325447/109728544-68423100-7b84-11eb-8fc0-759df7c3b974.png)

* Website: https://infrahq.com
* Docs: https://infrahq.com/docs
* Slack: https://infra-slack.slack.com

> We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)

## Introduction

Identity and access management for Kubernetes. Instead of creating separate credentials and writing scripts to map permissions to Kubernetes, developers & IT teams can integrate existing identity providers (Okta, Google accounts, GitHub auth, Azure active directory) to securely provide developers with access to Kubernetes.

Use cases:
* Fine-grained permissions
* Mapping existing users & groups (in Okta, Azure AD, Google, etc) into Kubernetes groups
* On-boarding and off-boarding users (automatically sync users against identity providers)
* No more out of sync Kubeconfig
* Cloud vendor-agnostic
* Audit logs (who did what, when)


## Architecture

![architecture](https://user-images.githubusercontent.com/3325447/110035578-da904e00-7d09-11eb-9546-eef4145fac27.png)

## Installing

Install Infra via `kubectl`:

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/kubernetes/infra.yaml
```

Then find the service on which Infra is listening:

```
$ kubectl get svc -n infra
NAME             TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra-engine    LoadBalancer   10.12.11.116   32.71.121.168   80:32322/TCP   1m
```

For users wishing to use infra-engine through a VPC or ingress, please see advanced set-up. 

Next, optionally map your dns (`infra.acme.com` in our example) to this domain via your DNS provider.

## Using Infra

### Installing the CLI

**Mac:** 

```
brew cask install infra
```

**Windows:** 

```
winget install --id infra.infra
```  

**Linux:** 

```
sudo curl -L "https://infrahq.com/download/linux-$(uname -m)" -o /usr/local/bin/infra

sudo chmod +x /usr/local/bin/infra
```

```
$ infra
Infra: manage Kubernetes access

Usage:
  infra [command]
  infra [flags]

Available Commands:
  help          Help about any command
  users         List all users across all groups
  groups        List available groups
  roles         List available roles
  permissions   List configured permissions
  login         Login to an Infra server
  logout        Log out of an Infra server
  install       Install infra-engine on a target Kubernetes cluster

Flags:
  -h, --help   help for infra

Use "infra [command] --help" for more information about a command.
```

### Login 

Run `infra login` to log into the infra server via the CLI

```
$ infra login infra.acme.com
... Opening Google login URL...

✓ Logged in
✓ Kubeconfig updated
```

Infra has updated your Kubeconfig with an entry for connecting to the cluster 

## Administration

### Listing users

List users that have been synchronized to Infra:

```
$ infra users
USER                 PROVIDER             ROLES            NAMESPACE
jeff@acme.com        google               admin            default
```

### Listing access 

List the user's access permissions

```
$ infra permissions -u jeff@acme.com

NAME                                                          LIST  CREATE  UPDATE  DELETE
alertmanagers.monitoring.coreos.com                           ✔     ✔       ✔       ✔
apiservices.apiregistration.k8s.io                            ✔     ✔       ✔       ✔
backups.velero.io                                             ✔     ✔       ✔       ✔
backupstoragelocations.velero.io                              ✔     ✔       ✔       ✔
bgpconfigurations.crd.projectcalico.org                       ✔     ✔       ✔       ✔
bindings                                                            ✔               
certificatesigningrequests.certificates.k8s.io                ✔     ✔       ✔       ✔
clusterinformations.crd.projectcalico.org                     ✔     ✔       ✔       ✔
clusterrolebindings.rbac.authorization.k8s.io                 ✔     ✔       ✔       ✔
clusterroles.rbac.authorization.k8s.io                        ✔     ✔       ✔       ✔
componentstatuses                                             ✔                     
configmaps                                                    ✔     ✔       ✔       ✔
controllerrevisions.apps                                      ✔     ✔       ✔       ✔
cronjobs.batch                                                ✔     ✔       ✔       ✔
csidrivers.storage.k8s.io                                     ✔     ✔       ✔       ✔
csinodes.storage.k8s.io                                       ✔     ✔       ✔       ✔
customresourcedefinitions.apiextensions.k8s.io                ✔     ✔       ✔       ✔
daemonsets.apps                                               ✔     ✔       ✔       ✔
daemonsets.extensions                                         ✔     ✔       ✔       ✔
deletebackuprequests.velero.io                                ✔     ✔       ✔       ✔
deployments.apps                                              ✔     ✔       ✔       ✔
deployments.extensions                                        ✔     ✔       ✔       ✔
downloadrequests.velero.io                                    ✔     ✔       ✔       ✔
endpoints                                                     ✔     ✔       ✔       ✔
events                                                        ✔     ✔       ✔       ✔
events.events.k8s.io                                          ✔     ✔       ✔       ✔
felixconfigurations.crd.projectcalico.org                     ✔     ✔       ✔       ✔
globalnetworkpolicies.crd.projectcalico.org                   ✔     ✔       ✔       ✔
globalnetworksets.crd.projectcalico.org                       ✔     ✔       ✔       ✔
horizontalpodautoscalers.autoscaling                          ✔     ✔       ✔       ✔
hostendpoints.crd.projectcalico.org                           ✔     ✔       ✔       ✔
ingresses.extensions                                          ✔     ✔       ✔       ✔
ingresses.networking.k8s.io                                   ✔     ✔       ✔       ✔
ippools.crd.projectcalico.org                                 ✔     ✔       ✔       ✔
jobs.batch                                                    ✔     ✔       ✔       ✔
leases.coordination.k8s.io                                    ✔     ✔       ✔       ✔
limitranges                                                   ✔     ✔       ✔       ✔
localsubjectaccessreviews.authorization.k8s.io                      ✔               
mutatingwebhookconfigurations.admissionregistration.k8s.io    ✔     ✔       ✔       ✔
namespaces                                                    ✔     ✔       ✔       ✔
networkpolicies.crd.projectcalico.org                         ✔     ✔       ✔       ✔
networkpolicies.extensions                                    ✔     ✔       ✔       ✔
networkpolicies.networking.k8s.io                             ✔     ✔       ✔       ✔
nodes                                                         ✔     ✔       ✔       ✔
nodes.metrics.k8s.io                                          ✔                     
persistentvolumeclaims                                        ✔     ✔       ✔       ✔
persistentvolumes                                             ✔     ✔       ✔       ✔
poddisruptionbudgets.policy                                   ✔     ✔       ✔       ✔
pods                                                          ✔     ✔       ✔       ✔
pods.metrics.k8s.io                                           ✔                     
podsecuritypolicies.extensions                                ✔     ✔       ✔       ✔
podsecuritypolicies.policy                                    ✔     ✔       ✔       ✔
podtemplates                                                  ✔     ✔       ✔       ✔
podvolumebackups.velero.io                                    ✔     ✔       ✔       ✔
podvolumerestores.velero.io                                   ✔     ✔       ✔       ✔
priorityclasses.scheduling.k8s.io                             ✔     ✔       ✔       ✔
prometheuses.monitoring.coreos.com                            ✔     ✔       ✔       ✔
prometheusrules.monitoring.coreos.com                         ✔     ✔       ✔       ✔
replicasets.apps                                              ✔     ✔       ✔       ✔
replicasets.extensions                                        ✔     ✔       ✔       ✔
replicationcontrollers                                        ✔     ✔       ✔       ✔
resourcequotas                                                ✔     ✔       ✔       ✔
resticrepositories.velero.io                                  ✔     ✔       ✔       ✔
restores.velero.io                                            ✔     ✔       ✔       ✔
rolebindings.rbac.authorization.k8s.io                        ✔     ✔       ✔       ✔
roles.rbac.authorization.k8s.io                               ✔     ✔       ✔       ✔
runtimeclasses.node.k8s.io                                    ✔     ✔       ✔       ✔
schedules.velero.io                                           ✔     ✔       ✔       ✔
secrets                                                       ✔     ✔       ✔       ✔
selfsubjectaccessreviews.authorization.k8s.io                       ✔               
selfsubjectrulesreviews.authorization.k8s.io                        ✔               
serverstatusrequests.velero.io                                ✔     ✔       ✔       ✔
serviceaccounts                                               ✔     ✔       ✔       ✔
services                                                      ✔     ✔       ✔       ✔
statefulsets.apps                                             ✔     ✔       ✔       ✔
storageclasses.storage.k8s.io                                 ✔     ✔       ✔       ✔
studyjobs.kubeflow.org                                        ✔     ✔       ✔       ✔
subjectaccessreviews.authorization.k8s.io                           ✔               
tfjobs.kubeflow.org                                           ✔     ✔       ✔       ✔
tokenreviews.authentication.k8s.io                                  ✔               
validatingwebhookconfigurations.admissionregistration.k8s.io  ✔     ✔       ✔       ✔
volumeattachments.storage.k8s.io                              ✔     ✔       ✔       ✔
volumesnapshotlocations.velero.io                             ✔     ✔       ✔       ✔
No namespace given, this implies cluster scope (try -n if this is not intended)
```


### Adding users 

Users can be added in 2 ways: 
-  infra.yaml for scriptability and integration into existing infrastructure as code tools such as Terraform, Ansible, Pulumi, and more. 
- [optional] manually add users: 

``` 
$ infra users add michael@acme.com --roles view --namespace default
User michael@acme.com added with the following permissions: 
USER                    PROVIDER             ROLES            NAMESPACE
michael@acme.com        local                view             default 

One-time password for login: 
$9fX5n@4l;3

Please login using:
infra login infra.acme.com
``` 

### Listing groups

To view groups that have been synchronized to Infra, use `infra groups`:

```
$ infra groups
NAME                  PROVIDER        USERS          ROLES
developers@acme.com   google          1              admin
local                 local           1              view
```

### Listing roles

To view all roles in the cluster, use `infra roles`:

```
$ infra roles
NAME        NAMESPACE           GRANTED GROUPS      GRANTED USERS        DESCRIPTION 
admin       default             1                   1                    Admin access
view        default             1                   1                    Read-only access
```


### Accessing the dashboard

Infra's dashboard is always available at `https://<infra hostname>/dashboard`

To view the ui, run `infra ui`. You'll automatically be logged if you're logged in on the CLI. Otherwise you'll be greeted with a login screen.

![product](https://user-images.githubusercontent.com/3325447/110035290-779eb700-7d09-11eb-952b-f18190a1ddb3.png)


## Advanced (Coming Soon)
* Adding additional Kubernetes clusters
* Auditing access/logs (when & who did what )

### Configuring Infra to be scripted 

Create a configuration file:

```yaml
domain: infra.acme.com

identity:
  providers:
    - name: google
      kind: oidc
      config: 
        client-id: acme-12345678.apps.googleusercontent.com
        client-secret: /etc/infra/client-secret
        issuer-url: https://accounts.google.com
        redirect-url: https://infra.acme.com:3090/v1/oidc/callback
        scope: ['https://www.googleapis.com/auth/admin.directory.group.readonly', 'openid', 'email']
      groups:
        - developers@acme.com

permissions:
  - provider: google
    group: developers@acme.com
    role: admin
    namespace: default            # optional namespace
```
