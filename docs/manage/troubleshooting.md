---
title: Troubleshooting
position: 6
---

# Troubleshooting Infra

## I am seeing a lot of TLS Handshake errors in my logs

Although the Infra connector will install with a minimum of settings in the Helm values file, you will also need to set the health check options. Refer to the [Kubernetes Connector](./connectors/kubernetes) installation page for more.

## I added a connector, but it's taking a long time to establish a connection

Depending on which cloud provider is hosting your cluster, it may take a few minutes for a Load Balancer to be configured.
