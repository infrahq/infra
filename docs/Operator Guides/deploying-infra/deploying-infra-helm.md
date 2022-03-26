# Kubernetes (Helm)


> Note: Infra uses [Secrets](./docs/secrets.md) to securely load secrets.
> It is _not_ recommended to use plain text secrets. Considering using another supported secret type.

> Please follow [Okta Configuration](./docs/providers/okta.md) to obtain `clientID` and `clientSecret` for connecting Okta to Infra.

```yaml
# example infra.yaml

# Add an Identity Provider
# Only Okta is supported currently
providers:
  - name: Okta
    url: example.okta.com
    clientID: example_jsldf08j23d081j2d12sd
    clientSecret:  example_plain_secret #see note above

grants:
# 1. Grant user(s) or group(s) as Infra administrator
# Setup an user as Infra administrator
  - user: you@example.com
    role: admin
    resource: infra

# 2. Grant user(s) or group(s) access to a resources
# Example of granting access to an individual user the `cluster-admin` role. The name of a resource is specified when installing the Infra Engine at that location.
  - user: you@example.com
    role: cluster-admin                  # cluster_roles required
    resource: kubernetes.example-cluster # limit access to the `example-cluster` Kubernetes cluster

# Example of granting access to an individual user through assigning them to the 'edit' role in the `web` namespace.
# In this case, Infra will automatically scope the access to a namespace.
  - user: you@example.com
    role: edit                               # cluster_roles required
    resource: kubernetes.example-cluster.web # limit access to only the `web` namespace in the `example-cluster` Kubernetes cluster

# Example of granting access to a group the `view` role.
  - group: Everyone
    role: view                           # cluster_roles required
    resource: kubernetes.example-cluster # limit access to the `example-cluster` Kubernetes cluster
```

### Step 3: Install Infra

```bash
helm repo add infrahq https://helm.infrahq.com/
helm repo update
helm upgrade --install infra infrahq/infra --set-file server.config.import=infra.yaml
```

Infra can be configured using Helm values. To see the available configuration values, run:

```bash
helm show values infrahq/infra
```

### Step 4: Login to Infra

Next, you'll need to find the URL of the Infra server to login to Infra.

#### Port Forwarding

Kubernetes port forwarding can be used in access the API server.

```bash
kubectl -n infrahq port-forward deployments/infra-server 8080:80 8443:443
```

Infra API server can now be accessed on `localhost:8080` or `localhost:8443`

#### LoadBalancer

Change the Infra server service type to `LoadBalancer`.

```bash
kubectl -n infrahq patch service infra-server -p '{"spec": {"type": "LoadBalancer"}}'
```

Note: It may take a few minutes for the LoadBalancer endpoint to be assigned. You can watch the status of the service with:

```bash
kubectl -n infrahq get service infra-server -w
```

Once the endpoint is ready, get the Infra API server URL.

```bash
kubectl -n infrahq get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```

#### Ingress

Follow the [Ingress documentation](./docs/helm.md#advanced-ingress-configuration) to configure your Infra server with a Kubernetes ingress.
Once configured, get the Infra API server URL.

```bash
kubectl -n infrahq get ingress infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```

#### API Server Access Key

If not provided by the user during Helm install, the admin access key will be randomly generated. Retrieve it using `kubectl`.

WARNING: This admin access key grants full access to Infra. Do not share it.

```bash
kubectl -n infrahq get secret infra-admin-access-key -o jsonpath='{.data.access-key}' | base64 -d
```

Once you have access to the Infra API server and the access key, login to Infra from the terminal.

```bash
infra login <INFRA_API_SERVER>
```

### Step 5: Access the Cluster

In order to get access to the cluster, the engine service must be accessible externally. The easiest way to achieve this is to use a LoadBalancer service.

```bash
kubectl -n infrahq patch service infra-engine -p '{"spec": {"type": "LoadBalancer"}}'
```

Switch to the cluster with Infra CLI.

```bash
infra use kubernetes.example_cluster
```

## Next Steps

### Connect Additional Kubernetes Clusters

Using Infra CLI:

Generate the helm install command via
```
infra destinations add kubernetes example-cluster
```

Run the output Helm command on the Kubernetes cluster to be added.

Example:
```
helm upgrade --install infra-engine infrahq/engine --set config.accessKey=2pVqDSdkTF.oSCEe6czoBWdgc6wRz0ywK8y --set config.name=kubernetes.example-cluster --set config.server=https://infra.acme.com
```

### Upgrade Infra

```
helm repo update
helm upgrade infra infrahq/infra --set-file server.config.import=infra.yaml
```
