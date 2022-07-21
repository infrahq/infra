---
title: Quickstart
position: 2
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

## Access the Infra Dashboard

Next, visit the Infra Dashboard. To retrieve the hostname, run:

```
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}" -w
```

{% callout type="info" %}

Note: it may take a few minutes for the LoadBalancer to be provisioned.

{% /callout %}

## Login via Infra CLI

```
infra login <infra hostname>
```

{% callout type="info" %}
You may be prompted to verify the fingerprint of the server's TLS certificate. The fingerprint can be found in the server logs:

```
kubectl logs --tail=-1 -l 'app.kubernetes.io/name=infra-server' | grep fingerprint
```

If you're not using Docker Desktop, you'll be need to specify a different endpoint than `localhost`. This endpoint can be found via the following `kubectl` command:

```
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}" -w
```

Note: it may take a few minutes for the LoadBalancer to be provisioned.

If your load balancer does not have a hostname (often true for GKE and AKS clusters), Infra will not be able to automatically create a TLS certificate for the server. On GKE you can use the hostname `<LoadBalancer IP>.bc.googleusercontent.com` instead of `localhost`.

Otherwise you'll need to configure the LoadBalancer with a static IP and hostname (see
[GKE docs](https://cloud.google.com/kubernetes-engine/docs/tutorials/configuring-domain-name-static-ip), or
[AKS docs](https://docs.microsoft.com/en-us/azure/aks/static-ip#create-a-static-ip-address)).
Alternatively you can use the `--skip-tls-verify` with `infra login`, or setup your own TLS certificates for Infra.

{% /callout %}

## Connect a Kubernetes cluster

Download the CA certificate that was generated for Infra and save it to a file. This certificate will be used by the CLI and by connectors to establish secure TLS communication:

```
kubectl get secrets/infra-server-ca --template='{{index .data "ca.crt"}}' | base64 --decode > infra.ca
```

Generate a connector key (note: this key is valid for 30 days, but can be extended via `--ttl`):

```
infra keys add connector
```

Next, use this access key to connect your cluster via `helm`:

```
helm upgrade --install infra-connector infrahq/infra \
  --set connector.config.name=example-cluster \
  --set connector.config.server=localhost \
  --set connector.config.accessKey=<CONNECTOR_KEY> \
  --set-file connector.config.serverTrustedCertificate=infra.ca
```

{% callout type="info" %}
It may take a few minutes for the cluster to connect. You can verify the connection by running `infra destinations list` and by looking at the connector logs:

```
kubectl logs -l 'app.kubernetes.io/name=infra-connector'
```

{% /callout %}

## Next Steps

- [Connect Okta](../identity-providers/okta.md) to onboard & offboard your team automatically
- [Manage & revoke access](../configuration/granting-access.md) to users or groups
- [Customize](../reference/helm-reference.md) your install with `helm`
