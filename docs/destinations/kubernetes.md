## Introduction

## Contents

* [Introduction](#introduction)
* [Contents](#contents)
* [Connect](#connect)

## Connect

[![helm](https://img.shields.io/badge/docs-helm-green?logo=bookstack&style=flat)](./docs/helm.md)

Registry host and API key needs to be retrieved from the main cluster.

In your main cluster context:

```
REGISTRY_HOST=$(kubectl get service -l infrahq.com/component=registry -o jsonpath="{.items[].status.loadBalancer.ingress[]['ip', 'hostname']}")
REGISTRY_TOKEN=$(kubectl get secret infra-engine -o jsonpath='{.data.engine-key}' | base64 --decode)
```

```yaml
# values.yaml
---
global:
  registry:
    enabled: false

engine:
  registry: $REGISTRY_HOST
  apiKey: $REGISTRY_KEY
```

In your new cluster context:

```
helm install --repo https://helm.infrahq.com/ --set global.registry.enabled=false infra infra
```

## Configuring Roles

<!--
TODO
-->
