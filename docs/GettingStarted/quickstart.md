---
title: Quickstart
category: 6125661f1d211d0010ef10a4
---

[block:image]
{
  "images": [
    {
      "image": [
        "https://files.readme.io/81c1b84-QuickStartHeader.png",
        "QuickStartHeader.png",
        1610,
        290,
        "#1a1325"
      ]
    }
  ]
}
[/block]

### Prerequisites: 
- Helm (v3+)
- Kubernetes cluster (v1.14+) (ie. Docker Desktop with Kubernetes or Minikube) 


### 1. Install Infra

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

This will output the Infra Access Key which you will use to login, please store this in a safe place as you will not see this again.


### 4. Connect the first Kubernetes cluster

```
infra destinations add kubernetes example-name
``` 

### 5. Create the first local user 

``` 
infra id add name@example.com 
```

### 6. Grant Infra administrator privileges to the first user

``` 
infra grants add -u name@example.com --role admin infra 
``` 

### 7. Grant Kubernetes cluster administrator privileges to the first user 
```
infra grants add -u name@example.com --role cluster-admin kubernetes.example-name
```

### 8. Login to Infra with the newly created user 

```
infra login 
``` 
Select the Infra instance, and login with username / password