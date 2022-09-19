---
title: Kubernetes
position: 1
---

# Kubernetes

## Connecting a cluster

{% tabs %}

{% tab label="Dashboard" %}

![Connect to a cluster](../../images/connectcluster.gif)

Navigate to the **Infrastructure** tab in the dashboard and click the **Connect cluster** button.

Enter a **Cluster name** in the text box.

Copy the commands shown. They will add the Helm repo and update it, and then install the Infra Connector onto the cluster. Ensure you are connected to the appropriate Kubernetes cluster and then paste the commands into your terminal to connect Infra to your cluster.

{% /tab %}

{% tab label="CLI"%}
First, generate an access key:

```
infra keys add connector
```

Next, use this access key to connect your cluster:

```
helm upgrade --install infra-connector infrahq/infra \
    --set connector.config.server=INFRA_SERVER_HOSTNAME \
    --set connector.config.accessKey=ACCESS_KEY \
    --set connector.config.name=example-cluster-name
```

{% /tab %}
{% /tabs %}

For more control over the Connector install, review the [Helm Reference](../../reference/helm.md).

Once you've connected a cluster, you can grant access via `infra grants add` or using the Dashboard. [Learn more about Grants in Infra](../grants.md).
