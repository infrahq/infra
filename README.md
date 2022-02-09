<p align="center">
  <img src="./docs/images/InfraGithub.png" />
</p>

**We take security very seriously.** If you believe you have found a security issue please report it to our security team by contacting us at security@infrahq.com.

## Introduction

Infra is **identity and access management** for your cloud infrastructure. It puts the power of fine-grained access to infrastructure like Kubernetes in your hands via existing identity providers such as Okta, Google Accounts, Azure Active Directory and more.

**Features**:
* Single-command access: `infra login`
* No more out-of-sync user configurations
* Fine-grained role assignment
* Onboard and offboard users via Okta (Active Directory, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)

## Quickstart

**Prerequisites:**
* Install [Helm](https://helm.sh/) (v3+)
* Install [Kubernetes](https://kubernetes.io/) (v1.14+)

### Step 1: Install Infra CLI

<details>
  <summary><strong>macOS</strong></summary>

  ```bash
  brew install infrahq/tap/infra
  ```

</details>

<details>
  <summary><strong>Windows</strong></summary>

  ```powershell
  scoop bucket add infrahq https://github.com/infrahq/scoop.git
  scoop install infra
  ```

</details>

<details>
  <summary><strong>Linux</strong></summary>

  ```bash
  # Ubuntu & Debian
  sudo echo 'deb [trusted=yes] https://apt.fury.io/infrahq/ /' >/etc/apt/sources.list.d/infrahq.list
  sudo apt update
  sudo apt install infra
  ```

  ```bash
  # Fedora & Red Hat Enterprise Linux
  sudo dnf config-manager --add-repo https://yum.fury.io/infrahq/
  sudo dnf install infra
  ```

</details>

### Step 2: Configure Infra YAML

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
helm upgrade --install -n infrahq --create-namespace infra infrahq/infra --set-file server.config.import=infra.yaml
```

Infra can be configured using Helm values. To see the available configuration values, run:

```bash
helm show values infrahq/infra
```

### Step 4: Login to Infra

Next, you'll need to find the URL of Infra Server to login to Infra.

#### Port Forwarding

Kubernetes port forwarding can be used in access the API server.

```bash
kubectl -n infrahq port-forward deployments/infra-server 8080:80 8443:443
```

Infra API server can now be accessed on `localhost:8080` or `localhost:8443`

#### LoadBalancer

Change the Infra Server service type to `LoadBalancer`.

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

Follow the [Ingress documentation](./docs/helm.md#advanced-ingress-configuration) to configure your Infra Server with a Kubernetes ingress.
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
infra destinations add kubernetes example-name
```

Run the output Helm command on the Kubernetes cluster to be added.

Example:
```
helm install infrahq/engine --set config.name=kubernetes.example-name --set config.accessKey=2pVqDSdkTF.oSCEe6czoBWdgc6wRz0ywK8y --set config.server=localhost --set config.skipTLSVerify=true
```

### Upgrade Infra

```
helm repo update
helm upgrade -n infrahq --create-namespace infra infrahq/infra --set-file server.config.import=infra.yaml
```

## [Security](./docs/security.md)

We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com).

## [Documentation](./docs)

* [Infra CLI Reference](./docs/cli.md)
* [Helm Chart Reference](./docs/helm.md)
* [Contributing](./docs/contributing.md)
* [License](./LICENSE)
