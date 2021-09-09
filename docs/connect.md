### Connect a Kubernetes cluster

First, switch to the cluster you installed the Infra Registry on, then retrieve your Registry and Api Key:

```
export INFRA_REGISTRY=$(kubectl get svc -n infrahq infra-registry -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}")
export INFRA_API_KEY=$(kubectl get secrets/infra-registry --template={{.data.defaultApiKey}} --namespace infrahq | base64 -D)
```

Next, switch to the cluster you want to add, then run:

```
helm install infra-engine infrahq/engine -n infrahq --set registry=$INFRA_REGISTRY --set apiKey=$INFRA_API_KEY
```

Run this command to connect an existing Kubernetes cluster. Note, this command can be re-used for multiple clusters or scripted via Infrastructure As Code (IAC).
