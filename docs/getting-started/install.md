---
title: Install Infra
position: 3
---

# Quickstart

## Prerequisites

- Install [helm](https://helm.sh/docs/intro/install/) (v3+)
- Kubernetes (v1.14+)

## Deploy Infra

Deploy Infra via `helm`:

```
helm repo add infrahq https://helm.infrahq.com/
helm repo update
helm install infra infrahq/infra
```

Next, visit the Infra Dashboard. To retrieve the URL, run:

```
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}" -w
```

Note: it may take a few minutes for the LoadBalancer to be provisioned.

## Setting up certificates

It is recommended to always run the Infra Server with a valid TLS certificate.

```md
{% tabs %}

{% tab label="Ingress" %}
Infra can be configured behind an ingress controller
{% /tab %}

{% tab label="Self-signed" %}
Windows instructions
{% /tab %}

{% /tabs %}
```

## Logging in via the CLI

## Next Steps

- [Configure Certificates](../configuration/certificates.md)
- [Connect Okta](../identity-providers/okta.md) to onboard & offboard your team automatically
- [Manage & revoke access](../configuration/granting-access.md) to users or groups
- [Customize](../reference/helm-reference.md) your install with `helm`
