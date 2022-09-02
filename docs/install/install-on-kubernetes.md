---
title: Install on Kubernetes
position: 2
---

# Install Infra on Kubernetes

## Prerequisites

- Install [Helm](https://helm.sh/) (v3+)
- Install [Kubernetes](https://kubernetes.io/) (v1.14+)

## Install via Helm

Create a `values.yaml` file to define the first user. Update the email address and password accordingly:

```yaml
server:
  config:
    users:
      - name: admin@example.com
        password: SetThisPassword! #note this password is now set as plaintext in this file

  # Create a "admin@example.com" user and set a password passed in as a file. The file will need
  # to be mounted into the pod using `volumes` and `volumeMounts`.
    # - name: admin@example.com
    #   password: file:/var/run/secrets/admin@example.com

  # Create an "admin@example.com" user and set a password passed in as an environment variable.
  # The environment variable will need to be injected into the pod using `env` or `envFrom`.
    # - name: admin@example.com
    #   password: env:ADMIN_PASSWORD

    grants:
      - user: admin@example.com
        role: admin
        resource: infra
```
Install Infra via `helm`:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm update --install infra infrahq/infra --values values.yaml
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
