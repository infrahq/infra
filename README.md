# Infra

Fast, secure team access for Kubernetes.

![infra](https://user-images.githubusercontent.com/251292/105530843-64cea680-5cb6-11eb-9d97-e3210ef79914.png)

* Website: https://infrahq.com
* Docs: https://infrahq.com/docs
* Slack: https://infra-slack.slack.com

## Introduction

Infra is a tool for providing centralized access to any Kubernetes cluster for any user.

Use cases:
* On-boarding and off-boarding Kubernetes users
* Mapping existing users & groups (in GSuite, Okta, etc) to cluster permissions
* Multi-cloud cluster access
* Accessing Kubernetes in CI/CD environments
* Build custom or third party tools on the Kubernetes API

Features:
* Single binary
* Easy to use UI & CLI
* Unified access point for any number of Kubernetes clusters
* User login via GitHub, Okta, Microsoft (more coming soon)
* Audit log
* Client `kubectl` integration
* Synchronize groups via GitHub, Okta, Microsoft AD
* Enforce RBAC rules across clusters
* RBAC based on common templates



## Get Started

### Installing

1. [Download](https://infrahq.com/download) the `infra` binary from our website

For easiest setup, Infra installs on an existing Kubernetes cluster.

```
$ infra install
...
creating serviceaccount
creating pvc
creating deployment
creating service
...
Installation complete, infra server running on https://35.192.192.29.

One time password generated for admin: 9kd1-19d8jajl4i10-2
```

### Logging in

Run `infra login` logs you into infra. If you're an admin, you'll be automatically logged in via cluster access.

### Viewing the UI

To view the ui, run `infra ui`. You'll automatically be logged if you're logged in on the CLI.

![Screen Shot 2021-01-22 at 2 40 10 PM](https://user-images.githubusercontent.com/251292/105537327-c1828f00-5cbf-11eb-9e8a-00b96678a121.png)

### Listing clusters

Infra automatically adds the installed cluster as a target cluster. To view clusters run:

```
$ infra clusters
NAME            ENDPOINT                    STATUS
default         https://35.192.192.29       Up
```

### Adding a user

By default, infra has no users:

```
$ infra users
No users.
```

The default identity provider for infra uses username/password to add users but this is configurable. See [identity providers]().

To add a user, run:

```
$ infra user add test@acme.com

User added, one-time password: m012ofj01281d2kla9

This password must be changed by the user on first login.
```

**Note:** this user is now available in Kubernetes as a user. It will work with existing RBAC.

```
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: view
  namespace: default
subjects:
- kind: User
  name: test@acme.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role #this must be Role or ClusterRole
  name: view
  apiGroup: rbac.authorization.k8s.io
```

### Accessing the cluster via Infra

To update your kubeconfig, run `infra update`. Infra automatically does this when running `infra login`. 

```
$ infra login 35.192.192.29 
Username: test@acme.com
Password: ***** <password from above> *****
Create a new password: *******************

Successfully logged in as test@acme.com
```

Your kubeconfig will now show a new server named `infra-default`:

```
$ kubectl config get-contexts
CURRENT   NAME                                                          CLUSTER                                                       AUTHINFO                                                      NAMESPACE
*         infra-default                                                 infra-default                                                 infra-default
          gke_test-central1-demo                                        gke_test-central1-demo                                        gke_test-central1-demo 
```

Kubectl will *just work* for this cluster.

```
$ kubectl get pods
NAME                             READY     STATUS    RESTARTS   AGE
infra-a0k29dk1-102dk12           1/1       Running   0          5m

$ kubectl describe pod/infra-a0k29dk1-102dk12
...
```

Since we're logged in as `test@acme.com`, this user can't delete the pod:
```
$ kubectl delete pod/infra-a0k29dk1-102dk12
Access denied.
```


### Viewing audit log

```
$ infra logs
USER                 ACTION           KIND            RESOURCE                    ALLOWED     AGE   
test@acme.com        LIST             POD                                         Y           1m
test@acme.com        GET              POD             infra-a0k29dk1-102dk12      Y           1m
test@acme.com        DELETE           POD             infra-a0k29dk1-102dk12      N           1m
```


### Giving user permissions

```
$ infra users
USER                PERMISSIONS               IDENTITY
test@acme.com       none                      password
```

```
$ infra permissions
NAME                DESCRIPTION               
view                Read-only access
edit                Read & write access
admin               Full cluster access                
```

```
$ infra grant test@acme.com admin
```

# Allow logging in via GitHub
$ infra auth enable github


$ infra auth 
```

### Security

We take security very seriously. If you have found a security vulnerability please disclose it privately and responsibly to us by email at [security@infrahq.com](mailto:security@infrahq.com)
