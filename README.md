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

```yaml
# example infra.yaml

# adding an Identity Provider 
# currently only Okta is supported
providers: 
  - name: Okta
    url: example.okta.com
    clientID: example_jsldf08j23d081j2d12sd 
    clientSecret:  example_plain_secret #see note above

grants:
# 1. Set up an initial user from IdP to become Infra administrator
  - user: you@example.com
    role: admin
    resource: infra
# Or set up an initial group of users from IdP to become Infra administrator
  - group: Admin  # case sensitive 
    role: admin 
    resource: infra 

# 2. Grant group(s) or user(s) from IdP to have access to the determined resource

# Example for granting access to an individual user the cluster admin role on a Kubernetes cluster named 'example-cluster'. This name is specified when installing Infra Engine. 

  - user: you@example.com 
    role: cluster-admin  #cluster_roles required
    resource: kubernetes.example-cluster # kubernetes cluster name 

# Example for granting access to an individual user the cluster role 'edit' on a namespace. In this case, Infra will automatically scope the cluster-role to a namespace. 

  - user: you@example.com
    role: edit  #cluster_roles required
    resource: kubernetes.example-cluster.web #specifying the 'web' namespace inside kubernetes cluster named 'example-cluster' 

# Example for granting access to a group called 'Everyone' from Okta to the Kubernetes cluster named 'example-cluster'. 
  - group: Everyone
    role: view  #cluster_roles required
    resource: kubernetes.example-cluster
```

### Step 3: Install Infra 


```bash
helm repo add infrahq https://helm.infrahq.com/

helm install -n infrahq --create-namespace infra infrahq/infra --set-file config.import=infra.yaml
```

You'll need the Infra Root API Token to log into Infra. Please generate this token by running the following commands: 

```
ROOT_API_TOKEN=$(kubectl -n infrahq get secrets infra -o jsonpath='{.data.root-api-token}' | base64 --decode)
echo $ROOT_API_TOKEN
```

**Please store this in a safe place.** 

Next, you'll need to find the URL of Infra Server to log into Infra. 

<details>
  <summary><strong>Default (LoadBalancer)</strong></summary>
  Note: It may take a few minutes for the LoadBalancer endpoint to be assigned. You can watch the status of the service with: 

  ```bash
    INFRA_SERVER=$(kubectl -n infrahq get services -l infrahq.com/component=infra -o jsonpath="{.items[].status.loadBalancer.ingress[*]['ip', 'hostname']}")
    echo $INFRA_SERVER
  ```
</details>
<details>
  <summary><strong>Ingress</strong></summary>

  ```bash
  INFRA_SERVER=$(kubectl -n infrahq get ingress -l infrahq.com/component=infra -o jsonpath="{.items[].status.loadBalancer.ingress[*]['ip', 'hostname']}")
  ```

</details>

<details>
  <summary><strong>ClusterIP</strong></summary>

  ```bash
  CONTAINER_PORT=$(kubectl -n infrahq get services -l infrahq.com/component=infra -o jsonpath="{.items[].spec.ports[0].port}")
  kubectl -n infrahq port-forward services infra 8080:$CONTAINER_PORT &
  INFRA_SERVER='localhost:8080'
  ```
</details>

From the terminal login to Infra 

```bash
infra login `URL` 
``` 


## Next Steps

### Connect Additional Kubernetes Clusters

Using Infra CLI: 

Generate the helm install command via 
```
infra destination add kubernetes example-name
``` 

Run the output Helm command on the Kubernetes cluster to be added. 

Example: 
```
helm install infrahq/engine --set infra.name=kubernetes.example-name --set infra.apiToken=2pVqDSdkTF.oSCEe6czoBWdgc6wRz0ywK8y --set infra.host=localhost --set infra.skipTLSVerify=true
```

### Upgrade Infra

```
helm repo update

helm upgrade -n infrahq --create-namespace infra infrahq/infra --set-file config.import=infra.yaml
```

## [Security](./docs/security.md)

We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com).

## [Documentation](./docs)

* [API Reference](./docs/api.md)
* [Infra CLI Reference](./docs/cli.md)
* [Helm Chart Reference](./docs/helm.md)
* [Contributing](./docs/contributing.md)
* [License](./LICENSE)
