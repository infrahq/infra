# Infra
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

> We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)

## Introduction

Identity and access management for Kubernetes. Instead of creating separate credentials and writing scripts to map permissions to Kubernetes, developers & IT teams can integrate existing identity providers (Okta, Google accounts, GitHub auth, Azure active directory) to securely provide developers with access to Kubernetes.

Use cases:
* Fine-grained permissions
* Mapping existing users & groups (in Okta, Azure AD, Google, etc) into Kubernetes groups
* On-boarding and off-boarding users (automatically sync users against identity providers)
* No more out of sync Kubeconfig
* Cloud vendor-agnostic
* Coming soon: Audit logs (who did what, when)


## Architecture

<p align="center">
  <br/>
  <br/>
  <img src="https://user-images.githubusercontent.com/251292/113448649-395cec00-93ca-11eb-9c70-ea4c5c9f82da.png" />
  <br/>
  <br/>
</p>

## Quick Start

1. Deploy Infra:

```
kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/kubernetes/infra.yaml
```

2. Install Infra CLI 
```
# macOS
brew cask install infra

# Windows
winget install --id infra.infra

# Linux
curl -L "https://github.com/infrahq/infra/releases/download/latest/infra-linux-$(uname -m)" -o /usr/local/bin/infra
```


3. Log into Infra 

```
infra login
```

## Infra CLI 

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
  login         Log in to an Infra engine
  logout        Log out of an Infra engine

Flags:
  -h, --help   help for infra

Use "infra [command] --help" for more information about a command.
```

## Administration

### Listing users

List users that have been added to Infra:

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

Please login using:
infra login --token

and provide the token:

-----BEGIN INFRA TOKEN-----
MzEuMjkuMTY4LjI5Omphc2RoMSExMGFzODEyIWo5MTBka2w6TFMwdExTMUNSVWRKVGlCRFJWSlVTVVpK
UTBGVVJTMHRMUzB0Q2sxSlNVUkxha05EUVdoTFowRjNTVUpCWjBsUlZVSnZTbVEyVUZaVk9HcHdlU3Ro
VG5SMlNsSldha0ZPUW1kcmNXaHJhVWM1ZHpCQ1FWRnpSa0ZFUVhZS1RWTXdkMHQzV1VSV1VWRkVSWGxT
YUUxNlRtMWFiVmw1V2xNeGJFNHlSVE5NVkZGNlRrUk5kRTlIUlhsWlV6RnFXbFJTYkZwcVZYcFBSMDEz
VG1wUmR3cElhR05PVFdwRmQwMTZUWGhOVkd0NVRVUkZORmRvWTA1TmFsbDNUWHBOZDAxcVFYbE5SRVUw
VjJwQmRrMVRNSGRMZDFsRVZsRlJSRVY1VW1oTmVrNXRDbHB0V1hsYVV6RnNUakpGTTB4VVVYcE9SRTEw
VDBkRmVWbFRNV3BhVkZKc1dtcFZlazlIVFhkT2FsRjNaMmRGYVUxQk1FZERVM0ZIVTBsaU0wUlJSVUlL
UVZGVlFVRTBTVUpFZDBGM1oyZEZTMEZ2U1VKQlVVTnVhR3czVEV3elpHeGFXa0ZuVkhWb2JVbDBlVEUw
WnpkcVZGRlZSRmh4Y0hWclRrd3paR1YyTHdwdFZHcEJUVEpEWTBGcVNETnlTMDkzWnpWVmVraDBRVE56
VTNKdmRtWkllRTVhUmtGRFpsbDRXbFZaY1VsRWJqRTRjakJXTDJKVGJIb3lkbUZuUjBSRENubEpjbXh1
Y0U1blpUUkZVelYyU25KUVdYRkpNRnBUVVZGblluZDZSV0ZpV0RWbmRHUlJNSFpoVG5WVGEwZ3pabGxH
UWxvMFZHcFpVV1ZDWkZFdlJWVUtZMkZHZUdkMGNuSnhkamRoTVhObU56TnZNbFJzVDJwUWQxWnFabmh1
TUdNNWNHOXpaMWhwUVVaaE1HeGxkMEpJTlVkbFJrUk9ZV2RxYzNKQ1JscFpVZ3BDVG1KUVVUbFplR1l5
YnpkNVNURTViRmhwTWpKQ05YQkRaa2RhTm1oUkx6bG5iV1JsYURjNVQwbFJORzFJUzFsT2JERlNiR2xI
T0dsdEt6RktiakJGQ2t0VWVFSmFjemRYYW1aTVZYRmtWMDVMVlVWWFNGZHRNR0pTZEVKbmRpdGxabWhW
Um5wNWJsSTNTVUpZUVdkTlFrRkJSMnBSYWtKQlRVRTBSMEV4VldRS1JIZEZRaTkzVVVWQmQwbERRa1JC
VUVKblRsWklVazFDUVdZNFJVSlVRVVJCVVVndlRVSXdSMEV4VldSRVoxRlhRa0pVWlROQmVHOVFUbXMy
UmxWbGFRb3dObk5qUzFKdkwyWlphVnBIUkVGT1FtZHJjV2hyYVVjNWR6QkNRVkZ6UmtGQlQwTkJVVVZC
VDFoU1luSlNNek5JVWxnMk5FaEZTM3AxV0d4RFUzVkhDbTkyZVV4TllUWlVUelV4WlV0T1FVZHRWaXRN
YldGVGNHOVhkemh1UlRaa1pGcHRMMUJPZHpCRFduQjRRUzgwWjFOQ2NFZ3JZbWxHVDBnd01rWkJlaXNL
WjNKV1IyVjNMM1F5YjI1Vk5sZGxVbk53WWtkR1MyRlJUalp5VDBWVU5HeFZhMmcxVlU1U1pXRlFMM0ZG
UTA5cVlsQkhSM0JFUzFKMWMzaFlOM1prT1FwMWRVZHpMM1ZsZUVOV05ucFZkRU55YkRSb1ZGUjRSVU5V
YmtwM1JsUlVZVEpoWkRKS2FtcEJhblpNTlVacGNXMXdPRXczYVhGTmJYVlRXbnBJVW14cENtZFZiRE5H
Vlhaa1dXSm5NMmxPVFU0M1NFbzBkM0oyY1hneE0zUjRWek12UXpJelowNWphM1JHTWxKUVNIVlNaMUJM
YVZwbVoyUjVlV1pLYmxJelR6a0thVFYyY1c4dlNuZEJlbEZ1ZVdJd2VuVlNkRzlHY20xNVdWSm9SUzlL
TVVaeVRVZ3hRakZUWkN0elpEbDFhRXM0TkVSU1ZuTlNlQzgyV2pScVZFRTlQUW90TFMwdExVVk9SQ0JE
UlZKVVNVWkpRMEZVUlMwdExTMHRDZz09Cg==
-----END INFRA TOKEN-----
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

### Configuring Infra to be scripted 

Create a configuration file:

```yaml
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

### FAQ

Q: How do I find my Kubernetes cluster IP? 

A: You can determine the Kubernetes cluster IP address by running `kubectl cluster-info` 
