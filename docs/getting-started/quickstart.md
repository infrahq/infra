# Quickstart

In this quickstart we'll set up Infra to manage single sign-on to Kubernetes:
* Install Infra CLI & Infra Server
* Connect a Kubernetes cluster
* Create a user and grant them view (read-only) access to the cluster

### Prerequisites

* Install [helm](https://helm.sh/docs/intro/install/) (v3+)
* Install Kubernetes [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) (v1.14+)
* A Kubernetes cluster. For local testing we recommend [Docker Desktop](https://www.docker.com/products/docker-desktop/)

### 1. Install Infra CLI

The Infra CLI is used to connect to the Infra server.

<details>
  <summary><strong>macOS</strong></summary>

  ```bash
  brew install infrahq/tap/infra
  ```

  You may need to perform `brew link` if your symlinks are not working.
  ```bash
  brew link infrahq/tap/infra
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
  echo 'deb [trusted=yes] https://apt.fury.io/infrahq/ /' | sudo tee /etc/apt/sources.list.d/infrahq.list
  sudo apt update
  sudo apt install infra
  ```
  ```bash
  # Fedora & Red Hat Enterprise Linux
  sudo dnf config-manager --add-repo https://yum.fury.io/infrahq/
  sudo dnf install infra
  ```
</details>


### 2. Setup an Infra server

Deploy an Infra server to kubernetes using helm.

```
helm repo add infrahq https://helm.infrahq.com/
helm repo update
helm install infra infrahq/infra
```

Once the Infra server is deployed, login to the server to complete the setup. Start by finding the hostname for the Infra Server you just deployed:

```
# Find your Infra Server hostname
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```

> Note: It may take a few minutes for the LoadBalancer endpoint to be assigned. You can watch the status of the service with:
> ```bash
> kubectl get service infra-server -w
> ```

Login to the Infra Server and follow the instructions to create your admin account:

```
infra login <INFRA_SERVER_HOSTNAME> --skip-tls-verify
```


### 3. Connect your first Kubernetes cluster

Generate an access key named `key` to connect Kubernetes clusters:

```
infra keys add connector-key connector
```

Next, use this access key to connect your first cluster via `helm`. **Note:** this can be the same cluster used to install Infra Server in step 2.

* `connector.config.name`: choose a name for this cluster
* `connector.config.server`: the same hostname used for `infra login`
* `connector.config.accessKey`: the key created above via `infra keys add`


```
helm upgrade --install infra-connector infrahq/infra \
  --set connector.config.name=example \
  --set connector.config.server=<INFRA_SERVER_HOSTNAME> \
  --set connector.config.accessKey=<ACCESS_KEY> \
  --set connector.config.skipTLSVerify=true
```

| Note: it may take a few minutes for the cluster to connect. You can verify the connection by running `infra destinations list`

### 4. Add a user and grant access to the cluster

Next, add a user:

```
infra id add user@example.com
```

| Note: Infra will provide you a one-time password to use in step 5.

Grant this user read-only access to the Kubernetes cluster you just connected to Infra:

```
infra grants add user@example.com kubernetes.example --role view
```

### 5. Login as the example user:

Use the one-time password in the previous step to log in as the user. You'll be prompted to change the user's password since it's this new user's first time logging in.

```
infra login <INFRA_SERVER_HOSTNAME> --skip-tls-verify
```

Next, view this user's cluster access:

```
infra list
```

You should see the user has `view` access to the `example` cluster connected above.

Lastly, connect to the Kubernetes cluster and access it:

```
infra use kubernetes.example
```

```
# Works since the user has view access
kubectl get pods -A

# Does not work
kubectl create namespace test-namespace
```

Congratulations, you've:
* Installed Infra
* Connected your first cluster
* Created a user and granted them `view` access to the cluster

### Next Steps

* [Connect Okta](../guides/identity-providers/okta.md) to onboard & offboard your team automatically
* [Manage & revoke access](../guides/granting-access.md) to users or groups
* [Understand Kubernetes roles](../connectors/kubernetes.md#roles) for understand different access levels Infra supports for Kubernetes
* [Customize your install](../install/install-on-kubernetes.md)

