---
title: Granting Access
position: 4
---

# Granting Access

Grants apply roles that define access levels to users for a particular resource. In the case of Kubernetes, that resource could be the entire cluster or an individual namespace.

## Roles

Infra allows granting different levels of access via **roles**, such as `view`, `edit` or `admin`. Kubernetes includes a few roles by default and then installing the Infra Connector adds a few more useful roles. You can also add custom roles to your cluster. [You can read more about roles here](roles.md).

## Grant access

{% tabs %}
{% tab label="Dashboard" %}
Navigate to **Infrastructure** in the Dashboard. Choose the cluster you want to grant access to. Enter an email address in the text box. As you type, you will filter down the list of all users available in the dropdown. Select the user you want to grant access to. Now choose the role you want to grant to the user. Click the **+ Add** button to add the grant. This has granted the chosen role to the chosen user for the entire cluster.

![Grant access](../images/grantaccess.png)

To grant access to a particular namespace, select the **Namespaces** tab and then select a namespace. Then use the same procedure to grant access just to a particular namespace.
{% /tab %}

{% tab label="CLI" %}
To grant access, use `infra grants add`. The user you grant access to must already exist. [Learn about adding users here](users.md). To grant a user the `edit` role on a cluster named `staging` run:

```bash
infra grants add user@example.com staging --role edit
```

Note: the same command can be used to grant access to a group using the boolean `--group` flag:

```bash
infra grants add --group engineering staging --role edit
```

{% /tab %}
{% /tabs %}

## Revoking access

{% tabs %}
{% tab label="Dashboard" %}
Navigate to **Infrastructure** and choose the cluster you want to revoke access from. If revoking from a namespace, choose the namespace. Each user has a dropdown on the right where you can select a new role, or click **x Remove** at the bottom of the list.

![Revoke access](../images/revokeaccess.png)
{% /tab %}
{% tab label="CLI" %}
To revoke access, use `infra grants remove`:

```bash
infra grants remove user@example.com staging --role edit
```

{% /tab %}
{% /tabs %}

## Viewing access

{% tabs %}
{% tab label="Dashboard" %}
Navigate to **Infrastructure** and choose the cluster you want to view. Under **Access**, you will see a list of users that have access to the cluster. You can also change their roles from here. To see users that can access a namespace, select the **Namespace** tab and select the namespace. You will then see the list of users that have access to that namespace.
![View access](../images/viewaccess.png)
{% /tab %}
{% tab label="CLI" %}

```console
infra grants list
  USER                 ROLE     DESTINATION
  jeff@infrahq.com     edit     development
  michael@infrahq.com  view     production

  GROUP          ROLE      DESTINATION
  Engineering    edit      development.monitoring
  Engineering    view      production
  Design         edit      development.web
```

{% /tab %}
{% /tabs %}
