# Installing k0s with Infra

Getting k0s to work with Infra requires a few extra steps to get working.
Out of the box, k0s does not include a load balancer or persistent storage
which are required to run Infra.

We'll install:
  * [Metallb](https://docs.k0sproject.io/head/examples/metallb-loadbalancer/)
  * [OpenEBS](https://docs.k0sproject.io/head/storage/)

This guide assumes that you've already installed the Infra CLI. You
can check out the [Quickstart guide](https://infrahq.com/docs/getting-started/quickstart)
which contains directions on how to install the CLI.

### 1. Download k0s

`curl -sSLf https://get.k0s.sh | sudo sh`

1. Create a k0s config file

As the root user, run:

`k0s config create > /etc/k0s/k0s.yaml`

This will dump out a k0s config file which you will need to edit.

### 2. Edit /etc/k0s/k0s.yaml to include OpenEBS

Modify `/etc/k0s/k0s.yaml` and include the section:

```
spec:
  extensions:
    storage:
      type: openebs_local_storage
```

### 3. Install k0s

Run the command:

`sudo k0s install controller --single -c /etc/k0s/k0s.yaml`

If this is a multinode system, you can omit `--single` flag when you're doing the install, otherwise
the controller and the kubernetes node will be created on the same instance. Once you've installed
k0s, start it with the command:

`sudo k0s start`

### 4. Check that everything is running

First check to make certain that k0s is up and running. It may take a few moments before
everything is up.

`sudo k0s status`

Also, check to make certain that OpenEBS is working correctly. Run the command:

`sudo k0s kubectl get storageclass`

This will output something that looks like:
```
NAME               PROVISIONER        RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
openebs-device     openebs.io/local   Delete          WaitForFirstConsumer   false                  24s
openebs-hostpath   openebs.io/local   Delete          WaitForFirstConsumer   false                  24s
```


### 5. Export a kubeconfig

`sudo k0s kubectl view --raw`

Save this into `~/.kube/config` (make sure you're not writing over an existing kube config) and you can now
use `kubectl` directly. You'll need to do this to work with Infra.

### 6. Install the Metallb load balancer

```
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.10.2/manifests/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.10.2/manifests/metallb.yaml
```

### 7. Create a ConfigMap to set up your IP address pool

Create the file `metallb-configmap.yaml` which looks something like:
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
      - <ip-address-range-start>-<ip-address-range-stop>
```

You can use private IP addresses in the IP Address range fields, but you'll need to make certain that any IP addresses are routed correctly.
For example, if your network is using `192.168.1.1` to `192.168.1.254`, you can pick a small range of IPs inside of that network
(make certain that your router isn't handing those IP addresses out with DHCP).

Once you've created the file, apply it with:

`kubectl apply -f metallb-configmap.yaml`

### 8. Create an OpenEBS configuration file for Infra

This is a simple file which tells Kubernetes to use OpenEBS which we configured earlier.
We'll call this file `openebs-persistence.yaml`.

```
---
server:
  persistence:
    className: openebs-hostpath
```

### 9. Deploy Infra

Now that we've got k0s running with Metallb and OpenEBS, it's time to install Infra.
Run the commands:

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
helm install infra infrahq/infra --values openebs-persistence.yaml
```

### 10. Verify that Infra installed correctly

Check to make certain that the Infra service is working. To check the load balancer, use:

`kubectl get service infra-server`

This should output something that looks like:

```
NAME           TYPE           CLUSTER-IP       EXTERNAL-IP     PORT(S)                      AGE
infra-server   LoadBalancer   10.101.219.177   192.168.42.51   80:30223/TCP,443:30331/TCP   5m
```

Check the deployment with the command:

`kubectl get deployment infra-server`

This will display something like:

```
NAME           READY   UP-TO-DATE   AVAILABLE   AGE
infra-server   1/1     1            1           5m
```

You should now be able to login to the Infra server with the command

`infra login <EXTERNAL-IP> --skip-tls-verify`

Use the same external IP which was automatically assigned by the Metallb load balancer.

### 11. Follow the rest of the Quickstart guide

The rest of the steps to get Infra connected should be the same as a standard Kubernetes
installation.  Open up the [Quickstart guide](https://infrahq.com/docs/getting-started/quickstart)
and you can complete installing Infra from the section "Connect your first Kuberenetes cluster".


