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


### 1. Self-Host Infra

```
helm repo add infrahq https://helm.infrahq.com/
helm repo update
helm install infra infrahq/infra
```

### 2. Install Infra CLI 

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

### 3. Login to Infra

```
infra login localhost
```

This will output the Infra Access Key which you will use to login in cases of emergency recovery. Please store this in a safe place as you will not see this again.

<details>
  <summary><strong>Find the login URL if not using localhost</strong></summary><br />
  
**LoadBalancer**

```bash
kubectl patch service infra-server -p '{"spec": {"type": "LoadBalancer"}}'
```

Note: It may take a few minutes for the LoadBalancer endpoint to be assigned. You can watch the status of the service with:

```bash
kubectl get service infra-server -w
```

Once the endpoint is ready, get the Infra API server URL.

```bash
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```

**Ingress**

```bash
kubectl get ingress infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```

</details>


### 4. Connect the first Kubernetes cluster

This connects the first Kubernetes cluster to the self-hosted Infra. You can connect the same Kubernetes cluster that Infra is self-hosted on. 

```
infra destinations add kubernetes.example-name
``` 

Run the output helm command on the Kubernetes cluster you want to connect to Infra. 


### 5. Create the first local user 

``` 
infra id add name@example.com 
```

This creates a one-time password for the created user. 

### 6. Grant Infra administrator privileges to the first user

``` 
infra grants add --user name@example.com --role admin infra 
``` 

### 7. Grant Kubernetes cluster administrator privileges to the first user 

```
infra grants add --user name@example.com --role cluster-admin kubernetes.example-name
```

<details>
  <summary><strong>
Supported roles/cluster roles</strong></summary><br />
  
Infra supports cluster roles and roles within your Kubernetes environment including custom ones. For simplicity, you can use cluster roles, and scope it to a particular namespace via Infra. 
  
**Example applying a cluster role to a namespace:** 
  ```
  infra grants add --user name@example.com --role edit kubernetes.example-name.namespace
  ```
**Default available Cluster roles within Kubernetes:**
- **cluster-admin** <br /><br />
  Allows super-user access to perform any action on any resource. When 'cluster-admin' role is granted without specifying a namespace, it gives full control over every resource in the cluster and in all namespaces. When it is granted with a specified namespace, it gives full control over every resource in the namespace, including the namespace itself.<br />
- **admin** <br /><br />
  Allows admin access, intended to be granted within a namespace.
The admin role allows read/write access to most resources in the specified namespace, including the ability to create roles and role bindings within the namespace. This role does not allow write access to resource quota or to the namespace itself. This role also does not allow write access to Endpoints in clusters created using Kubernetes v1.22+. <br /><br />
- **edit** <br /><br />
  Allows read/write access to most objects in a namespace.
This role does not allow viewing or modifying roles or role bindings. However, this role allows accessing Secrets and running Pods as any ServiceAccount in the namespace, so it can be used to gain the API access levels of any ServiceAccount in the namespace. This role also does not allow write access to Endpoints in clusters created using Kubernetes v1.22+. <br /><br />
- **view** <br /><br />
  Allows read-only access to see most objects in a namespace. It does not allow viewing roles or role bindings.
This role does not allow viewing Secrets, since reading the contents of Secrets enables access to ServiceAccount credentials in the namespace, which would allow API access as any ServiceAccount in the namespace (a form of privilege escalation).
</details>


### 8. Login to Infra with the newly created user 

```
infra login 
``` 

Select the Infra instance, and login with username / password

### 9. Use your Kubernetes clusters

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

Note: this requires the user to have either admin or view permissions to Infra. 

An example to grant the permission:
infra grants add --user name@example.com --role view infra 
```
</details>

### 10. Share the cluster(s) with other developers 

To share access with Infra, developers will need to install Infra CLI, and be provided the login URL. If using local users, please share the one-time password. 


## [Security](./docs/security.md)

We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com).

## [Documentation](./docs)

* [Infra CLI Reference](./docs/cli.md)
* [Helm Chart Reference](./docs/helm.md)
* [Contributing](./docs/contributing.md)
* [License](./LICENSE)
