---
title: Quickstart
position: 2
---

# Quickstart


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

After logging in to the UI, navigate to **Clusters**. Click the **+ Cluster** button at the top right. Enter a name for the cluster and click **Next**. Copy the command shown in the UI and paste it into your terminal and press Enter to run the command. This will add the Kubernetes Connector.

## Next Steps

- [Customize](../reference/helm-reference.md) your install with `helm`
- [Connect Okta](../identity-providers/okta.md) (or another identity provider) to onboard & offboard your team automatically
