---
title: Logging in
position: 1
---

# Logging In

## Install Infra CLI

{% partial file="../partials/cli-install.md" /%}

## Login to Infra

```
infra login SERVER
```

## See what you can access

Run `infra list` to view what you have access to:

```
infra list
```

## Switch to the cluster context you want

```
infra use DESTINATION
```

## Use your preferred tool to run commands

```
# for example, you can run kubectl commands directly after the infra context is set
kubectl [command]
```
