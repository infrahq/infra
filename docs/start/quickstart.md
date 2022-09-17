---
title: Quick Start
position: 2
---

Infra is rolling out it's Software as a Service offering which makes it easier than ever to setup infrastructure access for your organization. This page will show you everything you need to do to get started.

## _the tl;dr version_

1. _[Signup to Create a New Organization](#signup-to-create-a-new-organization)_
2. _[Open Your Infra Dashboard](#open-your-infra-dashboard)_
3. _[Add a Kubernetes Cluster](#add-a-kubernetes-cluster)_
4. _[Add a User](#add-a-user)_
5. _Profit_

## Signup to Create a New Organization

{% callout type="info" %}

We are slowly rolling out our SaaS product. If you are interested in onboarding before we open it up completely, let us know. Until then you can [install the self-hosted version of the Infra Server](../reference/selfhosted.md).

{% /callout %}

## Open Your Infra Dashboard

After signing up for Infra, you should be automatically redirected to your Infra Dashboard. Be sure to also confirm your email address when you receive your introductory email.

![Open your Dashboard](../images/quickstart-opendashboard.png)

Now we need to do two things:

- Add User(s)
- Add Cluster(s)

[Click here to learn more about the Infra Dashboard](../using/dashboard.md).

### Add a Kubernetes Cluster

Let's start by adding our first cluster. To add a cluster you need to first have two prerequisites:

- Install [helm](https://helm.sh/docs/intro/install/) (v3+)
- Kubernetes (v1.14+)

Navigate to **Clusters** in your Dashboard. Click the **Connect cluster** button and provide a cluster name.

You will be given the contents of a **Helm values file**. Either download the file or copy the contents to add to your own values file.

Then run this set of commands to install the Infra Helm repo:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
```

Finally, run the command shown in the UI to install the connector. If you save the values file as something other than `values.yaml` you will need to change that part of the command.

{% callout type="info" %}

Note: it may take a few minutes for the LoadBalancer to be provisioned.

If your load balancer does not have a hostname (often true for GKE and AKS clusters), Infra will not be able to automatically create a TLS certificate for the server. On GKE you can use the hostname `<LoadBalancer IP>.bc.googleusercontent.com`.

Otherwise you'll need to configure the LoadBalancer with a static IP and hostname (see
[GKE docs](https://cloud.google.com/kubernetes-engine/docs/tutorials/configuring-domain-name-static-ip), or
[AKS docs](https://docs.microsoft.com/en-us/azure/aks/static-ip#create-a-static-ip-address)).
Alternatively you can use the `--skip-tls-verify` with `infra login`, or setup your own TLS certificates for Infra.

{% /callout %}

[Click here to learn more about Adding Clusters](../manage/connectors/kubernetes.md).

### Add a User

Navigate to Users in the Dashboard. Click the **+ User** button and enter the email of the user you wish to add. They will receive an email which will allow them to set their password and login to Infra.

Navigate back to **Clusters** and choose the cluster you added above. On the right side of the window, choose your user and select a role to assign to that user. Alternatively, you can click the plus sign next to the cluster name to see a list of namespaces and then click on one of the namespaces. Now if you choose a user and role, you will be setting access for just that namespace rather than the entire cluster.

{% callout type="info" %}

If you want to see your own roles in this list, refer to the [Using Roles](../manage/roles.md) page to see how to enable them.

{% /callout %}

[Click here to learn more about Adding Users](../manage/users.md).

Consider adding your organization's Identity Provider, such as [Okta](../manage/idp/okta) or Azure AD or Google.

## Install the CLI

The last step to actually signin to a cluster is to install the CLI. You can find the instructions to do this for your Operating System on the [Install Infra CLI page](install-infra-cli.md).

Then run `infra login <dashboard url>`. Now you can use all your usual tools to set the context and work with Kubernetes.
