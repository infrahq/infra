---
title: Quickstart
position: 2
---

# Quickstart

## Prerequisites

* Install [helm](https://helm.sh/docs/intro/install/) (v3+)
* Install Kubernetes [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) (v1.14+)
* A Kubernetes cluster. For local testing we recommend [Docker Desktop](https://www.docker.com/products/docker-desktop/)

## Install Infra CLI

{% partial file="../partials/cli-install.md" /%}

## Deploy Infra

Deploy Infra to your Kubernetes cluster via `helm`:

```
helm repo add infrahq https://helm.infrahq.com/
helm repo update
helm install infra infrahq/infra
```

Next, log into your instance of Infra to setup your admin account:

```
infra login localhost
```

{% callout type="info" %}
If you're not using Docker Desktop, you'll be need to specify a different endpoint than `localhost`. This endpoint can be found via the following `kubectl` command:

```
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}" -w
```

Note: it may take a few minutes for the LoadBalancer to be provisioned.

{% /callout %}

## Connect a Kubernetes cluster

Generate a connector key (note: this key is valid for 30 days, but can be extended via `--ttl`):

```
infra keys add connector
```

Next, use this access key to connect your first cluster via `helm`:

```
helm upgrade --install infra-connector infrahq/infra \
  --set connector.config.name=example-cluster \
  --set connector.config.server=localhost \
  --set connector.config.accessKey=<CONNECTOR_KEY> \
  --set connector.config.skipTLSVerify=true
```

{% callout type="info" %}
It may take a few minutes for the cluster to connect. You can verify the connection by running `infra destinations list`
{% /callout %}

## Add a user and grant cluster access

Next, add a user:

```
infra users add user@example.com
```

Grant this user read-only access to the Kubernetes cluster you just connected to Infra:

```
infra grants add user@example.com example-cluster --role view
```

## Login as the example user

Use the temporary password in the previous step to log in as the user. You'll be prompted to change the user's password since it's this new user's first time logging in.

```
infra login localhost
```

Next, view this user's cluster access. You should see the user has `view` access to the `example-cluster` cluster connected above:

```
infra list
```

Switch to this Kubernetes cluster:

```
infra use example-cluster
```

Verify that the user **can** view resources in the cluster:

```bash
kubectl get pods -A
```

Lastly, verify the user **cannot** create resources:

```bash
kubectl create namespace new
```

## Conclusion

Congratulations, you've:
* Installed Infra
* Connected your first cluster
* Created a user and granted them `view` access to the cluster

## Next Steps

* [Connect Okta](../identity-providers/okta.md) to onboard & offboard your team automatically
* [Manage & revoke access](../configuration/granting-access.md) to users or groups
* [Customize](../reference/helm-reference.md) your install with `helm`

