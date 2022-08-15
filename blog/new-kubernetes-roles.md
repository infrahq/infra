---
title: New Kubernetes roles for port-forward, exec and more
author: Matt Williams
date: 2022-06-10
---

![Kubernetes Roles in Infra](https://raw.githubusercontent.com/infrahq/blog/main/assets/img/InfraKubernetesRolesPortforwardExec.png)

This week we released version 0.13.3 of Infra which introduces a few new features and resolved a few issues.

### New Kubernetes roles

One thing we hear from a lot of users is that they know they can create roles in Kubernetes, but they wish there were a few more basic roles included out of the box. So we added a few default roles. After installing Infra, you will now see roles for `logs`, `exec`, and `port-forward`, in addition to the existing roles of `cluster-admin`, `edit`, and `view`. This should make everyone's lives much easier.

As we begin to support more OIDC providers, it's important for the API to show which provider a user comes from. But this information wasn't available. Now it is and will make using other providers more useful. We also added the ability to see which users are members of any given group in both the API and the CLI.

```
$ infra groups list
  NAME        USERS
  Everyone    matt@example.com, patrick@example.com,
              fred@example.com
  Test group  patrick@example.com
```

Finally in the Web UI, we added Providers. As with the CLI, Okta is supported as an OIDC provider. Other providers will be coming soon in both the UI and the CLI.

Find all the updates in [the GitHub repo changelog](https://github.com/infrahq/infra/blob/main/CHANGELOG.md). Look at releases 0.13.2 and 0.13.3.
