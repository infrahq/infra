---
title: Install on Kubernetes
position: 2
---

# Install Infra on Kubernetes

## Prerequisites

- Install [Helm](https://helm.sh/) (v3+)
- Install [Kubernetes](https://kubernetes.io/) (v1.14+)

## Install via Helm

Install Infra via `helm`:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm install infra infrahq/infra
```

Find your load balancer endpoint:

```
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```

Finally, open this URL in your browser to get started.

## Connecting clusters

To connect Kubernetes clusters to Infra, see the [Kubernetes connector](../connectors/kubernetes.md) guide.

## Customize your install

To customize your install via `helm`, see the [Helm Reference](../reference/helm-reference.md)
