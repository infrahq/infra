---
title: Deploy Infra
position: 2
---

# Deploy Infra

## Prerequisites

- Install [helm](https://helm.sh/docs/intro/install/) (v3+)
- Kubernetes (v1.14+)

## Install Infra via `helm`

### Create an admin password

First, create an admin password via `kubectl`:

```bash
kubectl create secret generic infra-admin-credentials \
  --from-literal=INFRA_ADMIN_PASSWORD='SetAPassword!'
```

### Write a `values.yaml` file

Create a `values.yaml` file to define the first user. Update the admin username accordingly:

```yaml
server:
  config:
    users:
      - name: admin@example.com # edit me
        password: env:INFRA_ADMIN_PASSWORD

    grants:
      - user: admin@example.com
        role: admin
        resource: infra

  envFrom:
    - secretRef:
        name: infra-admin-credentials
```

### Deploy Infra via `helm`

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm upgrade --install infra infrahq/infra --values values.yaml
```

## Access the Infra Dashboard

Next, visit the Infra Dashboard. To retrieve the hostname, run:

```
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}" -w
```

Visit this hostname in your browser to access the Infra Dashboard:

![welcome](../images/uilogin.png)

{% callout type="info" %}

Note: it may take a few minutes for the LoadBalancer to be provisioned.

If your load balancer does not have a hostname (often true for GKE and AKS clusters), Infra will not be able to automatically create a TLS certificate for the server. On GKE you can use the hostname `<LoadBalancer IP>.bc.googleusercontent.com` instead of `localhost`.

Otherwise you'll need to configure the LoadBalancer with a static IP and hostname (see
[GKE docs](https://cloud.google.com/kubernetes-engine/docs/tutorials/configuring-domain-name-static-ip), or
[AKS docs](https://docs.microsoft.com/en-us/azure/aks/static-ip#create-a-static-ip-address)).
Alternatively you can use the `--skip-tls-verify` with `infra login`, or setup your own TLS certificates for Infra.

{% /callout %}

## Next Steps

- [Customize](../reference/helm-reference.md) your install with `helm`
- [Connect Okta](../identity-providers/okta.md) (or another identity provider) to onboard & offboard your team automatically
