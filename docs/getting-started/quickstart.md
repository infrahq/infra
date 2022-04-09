# Quickstart

In this quickstart we'll set up Infra to manage single sign-on to Kubernetes.

Follow these steps to install and setup Infra on Kubernetes.

### Prerequisites

To use this quickstart guide you will need `helm` and `kubectl` installed.

* Install [helm](https://helm.sh/docs/intro/install/) (v3+)
* Install Kubernetes [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) (v1.14+)

You will also need a Kubernetes cluster.


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

Once the Infra server is deployed, login to the server to complete the setup.

```
infra login INFRA_URL --skip-tls-verify
```

Use the following command to find the Infra login URL. If you are not using a `LoadBalancer` service type, see the [Install on Kubernetes Guide](../install/kubernetes.md) for more information.

> Note: It may take a few minutes for the LoadBalancer endpoint to be assigned. You can watch the status of the service with:
> ```bash
> kubectl get service infra-server -w
> ```

```bash
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```

This will output the admin access key which you can use to login in cases of emergency recovery. Please store this in a safe place as you will not see this again.


### 3. Setup a local user

Now that the Infra server is setup you can create a user.  The `infra id add` command creates a one-time password for the user.

```
infra id add name@example.com
```

Grant the user Infra administrator privileges.

```
infra grants add --user name@example.com --role admin infra
```

Grant the user Kubernetes cluster administrator privileges.

```
infra grants add --user name@example.com --role cluster-admin kubernetes.example-name
```

> To view different roles allowed for Kubernetes clusters, see [Kubernetes Roles](../connectors/kubernetes.md#roles)


### 4. Login to Infra with the newly created user

Login again to switch from admin to your newly created user.

```
infra login
```

Select the Infra instance, and login with username `name@example.com`, and the password
from the previous step.

### 5. Connect your first Kubernetes cluster

In order to add connectors to Infra, you will need to set three pieces of information:

* `connector.config.name` is a name you give to identify this cluster. For the purposes of this Quickstart, the name will be `example-name`
* `connector.config.server` is the hostname or IP address the connector will use to communicate with the Infra server. This will be the same INFRA_URL value from step 2.
* `connector.config.accessKey` is the access key the connector will use to communicate with the server. You can use an existing access key or generate a new access key as shown below:

Generate an access key:

```
infra keys add KEY_NAME connector
```

Next, use this access key to connect your first cluster:

```bash
helm upgrade --install infra-connector infrahq/infra \
  --set connector.config.server=INFRA_URL \
  --set connector.config.accessKey=ACCESS_KEY \
  --set connector.config.name=example-name \
  --set connector.config.skipTLSVerify=true
```


### 6. Use your Kubernetes clusters

You can now access the connected Kubernetes clusters via your favorite tools directly. Infra in the background automatically synchronizes your Kubernetes configuration file (kubeconfig).

Alternatively, you can switch Kubernetes contexts by using the `infra use` command:

```
infra use kubernetes.example-name
```

<details>
  <summary><strong>Here are some other commands to get you started</strong></summary><br />

See the cluster(s) you have access to:
```
infra list
```
See the cluster(s) connected to Infra:
```
infra destinations list
```
See who has access to what via Infra:
```
infra grants list

Note: this requires the user to have the admin role within Infra.

An example to grant the permission:
infra grants add --user name@example.com --role admin infra
```
</details>

### 7. Share the cluster(s) with other developers

To share access with Infra, developers will need to install Infra CLI, and be provided the login URL. If using local users, please share the one-time password.
