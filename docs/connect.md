### Connect a Kubernetes cluster

First, switch to the cluster context where you installed the Infra Registry, then retrieve your Registry and Api Key:

```
export INFRA_REGISTRY=$(kubectl get svc -n infrahq infra-registry -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}")
export ENGINE_API_KEY=$(kubectl get secrets/infra-registry --template={{.data.engineApiKey}} --namespace infrahq | base64 -D)
```

Next, switch to the cluster you want to add:

```
kubectl config use-context <your other context name>
```

Finally, add the new cluster by installing the Infra Engine:

```
helm install infra-engine infrahq/engine -n infrahq --set registry=$INFRA_REGISTRY --set apiKey=$ENGINE_API_KEY
```

Run this command to connect an existing Kubernetes cluster. Note, this command can be re-used for multiple clusters or scripted via Infrastructure As Code (IAC).
