---
title: Using Infra as a User
position: 1
---

# Welcome to Infra

You may have been invited to Infra by your admin or just want to know more about what you can do with Infra. This guide will show you everything you need to know.

When you follow the link in the email, one of two things may happen.

- If your organization uses an Identity Provider such as Okta and has configured it with Infra, then you will be prompted to login.
- If you don't use an Identity Provider, or your administrator hasn't yet configured it, then you will be prompted to create a password.

Then you will be redirected to your Infra Dashboard. As a user all you can do here is see the clusters you have access to. Click on any cluster and then click the **Access cluster** button at the top right. This will show three commands:

```bash
infra login <your Infra Organization url>
infra use <your cluster name>
kubectl get pods
```

The first command is what you will use most of the time. The second command simply chooses your context from your `kubeconfig` file. You could just as easily use `kubectl config use-context <your cluster name>`. And then the third command represents anything you might do with your cluster.

## It looks simple, so why do we need it?

As an end-user, it may not be obvious what Infra is doing. When you login using the `infra login` command, your `kubeconfig` file is updated with the clusters, users, and contexts you are allowed to access as defined by your administrator. If you take a look at that file, you will see whatever was in there before, plus some additional entries added by `infra`. Behind the scenes, **Infra** is checking that you still should have access with each command you run. As soon as access is revoked for whatever reason, you won't be able to run commands against your cluster anymore. Similarly, if your access is increased, perhaps to deal with an incident, you will see that elevated access instantly.
