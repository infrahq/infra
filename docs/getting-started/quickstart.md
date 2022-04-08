## Quickstart

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

Use the following command to find the Infra login URL. If you are not using a `LoadBalancer` service type, see the [Deploy Kubernetes guide](../operator-guides/deploy/kubernetes.md) for more information.

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

<details>
  <summary><strong>Supported Kubernetes cluster roles</strong></summary><br />
  
Infra supports any cluster roles within your Kubernetes environment, including custom ones. For simplicity, you can use cluster roles, and scope it to a particular namespace via Infra. 
  
**Example applying a cluster role to a namespace:** 
  ```
  infra grants add --user name@example.com --role edit kubernetes.example-name.namespace
  ```
**Default cluster roles within Kubernetes:**
- **cluster-admin** <br /><br />
  Allows super-user access to perform any action on any resource. When the 'cluster-admin' role is granted without specifying a namespace, it gives full control over every resource in the cluster and in all namespaces. When it is granted with a specified namespace, it gives full control over every resource in the namespace, including the namespace itself.<br /><br />
- **admin** <br /><br />
  Allows admin access, intended to be granted within a namespace.
The admin role allows read/write access to most resources in the specified namespace, including the ability to create roles and role bindings within the namespace. This role does not allow write access to resource quota or to the namespace itself.<br /><br />
- **edit** <br /><br />
  Allows read/write access to most objects in a namespace.
This role does not allow viewing or modifying roles or role bindings. However, this role allows accessing Secrets and running Pods as any ServiceAccount in the namespace, so it can be used to gain the API access levels of any ServiceAccount in the namespace.<br /><br />
- **view** <br /><br />
  Allows read-only access to see most objects in a namespace. It does not allow viewing roles or role bindings.
This role does not allow viewing Secrets, since reading the contents of Secrets enables access to ServiceAccount credentials in the namespace, which would allow API access as any ServiceAccount in the namespace (a form of privilege escalation).
</details>


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
* `connector.config.accessKey` is the access key the connector will use to communicate with the server. You can use an existing access key or generate a new access key with `infra keys add KEY_NAME connector`

```bash
helm upgrade --install infra-connector infrahq/infra --set connector.config.server=INFRA_URL --set connector.config.accessKey=ACCESS_KEY --set connector.config.name=example-name --set connector.config.skipTLSVerify=true
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
