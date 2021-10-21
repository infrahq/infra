## Introduction

## Contents

* [Introduction](#introduction)
* [Contents](#contents)
* [Connect](#connect)

## Connect

Registry host and API key needs to be retrieved from the main cluster.

In your main cluster context:

```
REGISTRY_HOST=$(kubectl -n infrahq get service -l infrahq.com/component=registry -o jsonpath="{.items[].status.loadBalancer.ingress[]['ip', 'hostname']}")
REGISTRY_API_KEY=$(kubectl -n infrahq get secret infra-engine -o jsonpath='{.data.engine-key}' | base64 --decode)
```

In your new cluster context:

```
helm install -n infrahq --create-namespace --set registry=$REGISTRY_HOST --set apiKey=REGISTRY_API_KEY infra-engine infrahq.com/engine
```

To customize your install, see the [Helm Chart reference](./../helm.md).
