## Introduction

## Contents

* [Introduction](#introduction)
* [Contents](#contents)
* [Connect](#connect)

## Connect

Registry host and API key needs to be retrieved from the main cluster.

In your main cluster context:

```
REGISTRY_HOST=$(kubectl get service -l infrahq.com/component=registry -o jsonpath="{.items[].status.loadBalancer.ingress[]['ip', 'hostname']}")
REGISTRY_API_KEY=$(kubectl get secret infra-engine -o jsonpath='{.data.engine-key}' | base64 --decode)
```

In your new cluster context:

```
helm install --set registry=$REGISTRY_HOST --set apiKey=REGISTRY_API_KEY infra-engine infrahq.com/engine
```

[Helm Chart Reference](./helm.md)

## Configuring Roles

<!--
TODO
-->
