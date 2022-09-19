---
title: Self-hosted Infra
position: 5
---

# Self-hosted Infra

In most ways, self-hosted offers similar features to our Software as a Service offering. In fact, the main difference is that you host the "Server" on your own Kubernetes cluster. This document will show you how to install that server. Since the rest of the product is the same in the SaaS model and Self Hosted, you can refer to the rest of the documentation for everything else.

## Prerequisites

- Install [helm](https://helm.sh/docs/intro/install/) (v3+)
- Kubernetes (v1.14+)

## Deploy Infra

### Create an admin password

First, create a secret for your admin password:

```bash
kubectl create secret generic infra-admin-credentials \
--from-literal=INFRA_ADMIN_PASSWORD='SetAPassword!'
```

Create a `values.yaml` file to define the first user. Update the email address accordingly:

```yaml
server:
  config:
    users:
      - name: admin@example.com # edit me
        password: env:INFRA_ADMIN_PASSWORD

    # Create a "admin@example.com" user and set a password passed in as a file. The
    # file will need to be mounted into the pod using `volumes` and `volumeMounts`.
    # - name: admin@example.com
    #   password: file:/var/run/secrets/admin@example.com

    grants:
      - user: admin@example.com
        role: admin
        resource: infra

  envFrom:
    - secretRef:
        name: infra-admin-credentials
```

{% callout type="info" %}

This example shows two ways to use secrets in the values file. You can learn more about the Helm values file in the [Helm Reference](../reference/helm.md).

{% /callout %}

Deploy Infra via `helm`:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm upgrade --install infra infrahq/infra --values values.yaml
```

{% callout type="info" %}

In rare cases, we have seen `helm upgrade --install` not behave correctly. If you haven't installed Infra before and are seeing long delays, consider running the command with `helm install` instead.

{% /callout %}

Find your load balancer endpoint:

```
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```

Depending on where you are hosting your cluster, the creation of the load balancer can take 10 minutes or more by a cloud provider. If you want to leverage an existing load balancer for the server, refer to the [Helm Reference](../reference/helm.md).

Finally, open the endpoint in your browser to get started using the Infra UI.

## Connecting clusters

To connect Kubernetes clusters to Infra, see the [Kubernetes connector](../manage/connectors/kubernetes.md) guide.

## Customize your install

To customize your install via `helm`, see the [Helm Reference](../reference/helm.md)
