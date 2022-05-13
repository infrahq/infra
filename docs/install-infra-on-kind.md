# Installing Infra on kind

This guide will show you how to set up [kind](https://kind.sigs.k8s.io/) and Infra in Linux
using the Metallb load balancer. Docker Desktop for macOS and Windows both use a virtual machine
which runs Linux, and does not directly expose docker's network to the host. Consider using
Docker's built-in kubernetes implementation instead of kind if you are using Docker Desktop.

This guide assumes that you have already [installed kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/) and [docker](https://docs.docker.com/engine/install/).

### 1. Download the latest version of kind

You'll first need to download a copy of kind. We'll use one of the pre-compiled binaries,
however, you can also follow the [kind quickstart guide](https://kind.sigs.k8s.io/docs/user/quick-start/)
for alternate methods.

We're assuming that you are using an Intel 64 bit instance, but there are other releases available [here](https://github.com/kubernetes-sigs/kind/releases/).

```
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.13.0/kind-linux-amd64
chmod +x ./kind
mv ./kind /some-dir-in-your-PATH/kind
```

### 2. Create a cluster

Once you have the kind binary configured in your path, create your kubernetes cluster using the command:

```
kind create cluster
```

### 3. Create a load balancer

Now that your kubernetes cluster is up, we'll install the Metallb load balancer.

```
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.12.1/manifests/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.12.1/manifests/metallb.yaml
```

You can verify that the load balancer is running with the command:

```
kubectl get pods -n metallb-system
```

### 4. Determine the IP address pool for Docker

We still need to set up the IP address pool which Metallb will use. Since we're using docker, we
need to look at the docker networking stack to determine which IP addresses we can use.


```
docker network inspect -f '{{.IPAM.Config}}' kind
```

This will print out something similar to:

```
[{172.19.0.0/16  172.19.0.1 map[]} {fc00:f853:ccd:e793::/64   map[]}]
```

In this case our docker network is set to use `172.19.0.0/16`, however it may be different on your
Linux host. We'll configure the load balancer to use IPs in the range `172.19.255.200-172.19.255.250`
since it's unlikely for Docker to assign any of those IPs to containers. 

Next, create a ConfigMap called `metallb-configmap.yaml` with IPs in docker's network range. It should look something like:

```
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: default
      protocol: layer2
      addresses:
      - 172.19.255.200-172.19.255.250
```

Once you're finished, apply the ConfigMap with the command:

`kubectl apply -f metallb-configmap.yaml`

### 5. Follow the rest of the Quickstart guide

The rest of the steps to get Infra connected should be the same as a standard Kubernetes
installation.  Open up the [Quickstart guide](https://infrahq.com/docs/getting-started/quickstart)
and you can complete installing Infra from the section "Install Infra CLI".

