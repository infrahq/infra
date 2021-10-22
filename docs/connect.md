### Connect a Kubernetes cluster

First, switch to the cluster context where you installed Infra, then retrieve your host and API Key:

```
INFRA_HOST=$(kubectl -n infrahq get services -l infrahq.com/component=infra -o jsonpath="{.items.status.loadBalancer.ingress[]['ip', 'hostname']}")
ENGINE_API_KEY=$(kubectl -n infrahq get secrets infra --template={{.data.engine-key}} | base64 -D)
```

Next, switch to the cluster you want to add:

```
kubectl config use-context <your other context name>
```

Finally, add the new cluster by installing the Infra Engine:

```
helm -n infrahq install infra-engine infrahq/engine --set host=$INFRA_HOST --set apiKey=$ENGINE_API_KEY
```

Run this command to connect an existing Kubernetes cluster. Note, this command can be re-used for multiple clusters or scripted via Infrastructure As Code (IAC).
