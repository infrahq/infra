# Infra

![infra](https://user-images.githubusercontent.com/251292/105530843-64cea680-5cb6-11eb-9d97-e3210ef79914.png)

* Website: https://infrahq.com
* Docs: https://infrahq.com/docs
* Slack: https://infra-slack.slack.com

> We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)

## Introduction

Identity and access management for Kubernetes. Instead of creating separate credentials and writing scripts to map permissions to Kubernetes, developers & IT teams can integrate existing identity providers (GitHub Auth, Okta, Microsoft AD, or Google accounts) to securely provide developers with access to Kubernetes.

Use cases:
* Fine-grained permissions
* Multi-cloud cluster access
* Mapping existing users & groups (in GSuite, Okta, etc) into Kubernetes groups
* On-boarding and off-boarding users


## Architecture

![Screen Shot 2021-01-27 at 11 05 44 PM](https://user-images.githubusercontent.com/251292/106088560-3573cb80-60f4-11eb-8f6f-8ae6688418f4.png)

## Installing

Install Infra via `kubectl`:

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/kubernetes/infra.yaml
```

Then find the service on which Infra is listening:

```
$ kubectl get svc -n infra
NAME             TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra-service    LoadBalancer   10.12.11.116   32.71.121.168   80:32322/TCP   1m
```

Next, optionally map your dns (`infra.acme.com` in our example) to this domain via your DNS provider.

## Using Infra

### Installing the CLI

* Mac: 

  `brew cask install infra`

* Windows: 

  `winget install --id infra.infra`  

* Linux: 

  `sudo curl -L "https://infrahq.com/download/linux-$(uname -m)" -o /usr/local/bin/infra`

  `sudo chmod +x /usr/local/bin/infra` 

```
$ infra
Infra: manage Kubernetes access

Usage:
  infra [command]
  infra [flags]

Available Commands:
  help        Help about any command
  users       List all users across all groups
  groups      List available groups
  roles       List available roles
  login       Login to an Infra server
  logout      Log out of an Infra server
  server      Run the infra server
  install     Install infra-engine on a target Kubernetes cluster

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
jeff@acme.com        google               view             default
```

### Adding users 

Users can be added in 2 ways: 
-  infra.yaml for scriptability and integration into existing infrastructure as code tools such as Terraform, Ansible, Pulumi, and more. 
- [optional] manually add users: 

``` 
$ infra users add michael@acme.com --roles view --namespace default
User michael@infrahq.com added with the following permissions: 
USER                    PROVIDER             ROLES            NAMESPACE
michael@acme.com        manual               view             default 

One-time password for login: 
$9fX5n@4l;3
infra login infra.acme.com
``` 




### Listing groups

To view groups that have been synchronized to Infra, use `infra groups`:

```
$ infra groups
NAME                  PROVIDER        USERS          ROLES
developers@acme.com   google          2              view
```

### Listing roles

To view all roles in the cluster, use `infra roles`:

```
$ infra roles
NAME        NAMESPACE           GRANTED GROUPS      GRANTED USERS        DESCRIPTION 
view        default             1                   2                    Read-only access
```

### Logging in

 using the password from the previous step:

> Note: make sure you log into a Google account that's part of the group you specified when configuring Infra.

```
$ infra login infra.acme.com
... Opening Google login URL...

✓ Logged in
✓ Kubeconfig updated
```

Infra has updated your Kubeconfig with an entry for connecting to the cluster via Infra:

```
$ kubectl get pods -A
kube-system   event-exporter-gke-564fb97f9-wvwrf                             2/2     Running   0          4d3h
kube-system   fluentbit-gke-5b49s                                            2/2     Running   0          4d3h
kube-system   fluentbit-gke-6f2xf                                            2/2     Running   0          4d3h
kube-system   gke-metrics-agent-h2crq                                        1/1     Running   0          4d3h
kube-system   gke-metrics-agent-w5xbj                                        1/1     Running   0          4d3h
kube-system   konnectivity-agent-h8wzm                                       1/1     Running   0          4d3h
kube-system   konnectivity-agent-vrrs4                                       1/1     Running   0          4d3h
kube-system   kube-dns-6bd88c9b66-j7jpj                                      4/4     Running   0          4d3h
kube-system   kube-dns-6bd88c9b66-qfwln                                      4/4     Running   0          4d3h
kube-system   kube-dns-autoscaler-7f89fb6b79-jr6dc                           1/1     Running   0          4d3h
kube-system   kube-proxy-gke-infra-app-production-production-6804f449-4jmx   1/1     Running   0          4d3h
kube-system   kube-proxy-gke-infra-app-production-production-6804f449-uriy   1/1     Running   0          4d3h
kube-system   l7-default-backend-5b76b455d-2lw7n                             1/1     Running   0          4d3h
kube-system   metrics-server-v0.3.6-7c5cb99b6f-kzcqr                         2/2     Running   0          4d3h
kube-system   stackdriver-metadata-agent-cluster-level-7d7947fd69-bxtxz      2/2     Running   0          4d3h
```

Since we specified view access to this user group, they cannot create or delete any resources:

```
$ kubectl run nginx --image=nginx
403 Forbidden
```

### Accessing the dashboard

Infra's dashboard is always available at `https://<infra hostname>/dashboard`

To view the ui, run `infra ui`. You'll automatically be logged if you're logged in on the CLI. Otherwise you'll be greeted with a login screen.

![Screen Shot 2021-01-22 at 2 40 10 PM](https://user-images.githubusercontent.com/251292/105537327-c1828f00-5cbf-11eb-9e8a-00b96678a121.png)


## Advanced

### Auditing access

### Adding additional Kubernetes clusters

## Coming Soon

* Configuration via yaml
* Support for GSuite, GitHub, Okta, Microsoft AD identity providers
* Groups (incl. sync from identity providers)
* Enforce RBAC rules across clusters
* UI
* Multiple clusters
* Dynamic cluster discovery
* Tunneling for cross-network access
* Support for Postgresql storage back-end
* Audit log
* Multi-namespace and multi-cluster queries

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
        client-secret: example-secret
        issuer-url: https://accounts.google.com
        redirect-url: https://infra.acme.com:3090/v1/oidc/callback
        scope: ['https://www.googleapis.com/auth/admin.directory.group.readonly', 'openid', 'email']
      groups:
        - developers@acme.com

permissions:
  - provider: google
    group: developers@acme.com
    role: view
    namespace: default            # optional namespace
```
