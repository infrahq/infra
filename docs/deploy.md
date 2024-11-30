# Deploy Infra

## Prerequisites

- Install [Helm](https://helm.sh/docs/intro/install/) (v3+)
- Kubernetes (v1.14+)

## Quickstart

### Deploy Infra

Deploy Infra via `helm`:

```shell
helm repo add infrahq https://infrahq.github.io/helm-charts
helm repo update
```

Create an initial `values.yaml`:

```
server:
  service:
    type: LoadBalancer
```

Deploy the Helm chart:

```
helm install infra-server infrahq/infra-server -f values.yaml
```

Find your admin credentials:

```shell
kubectl get secret infra-server-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d
```

Find your Load Balancer endpoint:

```shell
kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```

Depending on where you are hosting your cluster, the creation of the load balancer can take 10 minutes or more by a cloud provider.

### Log in

Download the [Infra CLI](./download) and log in with user `admin@local`:

```shell
export INFRA_SERVER=<your self-hosted server endpoint>
export INFRA_USER=admin@local
infra login $INFRA_SERVER
```

### Generate an access key

```shell
export INFRA_ACCESS_KEY=$(infra keys add --connector -q)
```

### Connect Kubernetes cluster

Install Infra connector via [helm](https://helm.sh):

```
helm install infra infrahq/infra --set config.server.url=$INFRA_SERVER --set config.accessKey=$INFRA_ACCESS_KEY --set config.name=example
```

> It may take a few minutes for your cluster to connect. To avoid TLS certificate errors, make sure you run Infra behind an [ingress](#ingress) controller below with certificates configured.

To finish, verify that your cluster is connected:

```
infra destinations list
```

> Infra supports multiple configuration methods for connecting Kubernetes clusters. Please follow the [Kubernetes](../integrations/kubernetes#connect) guide.

## Automated install with Terraform

This section outlines how to set up Infra programatically using the Infra [Terraform Provider](https://registry.terraform.io/providers/infrahq/infra).

Start by adding the Helm chart:

```
helm repo add infrahq https://infrahq.github.io/helm-charts
helm repo update
```

Generate a pre-defined admin access key:

```
export INFRA_ACCESS_KEY="$(openssl rand -hex 5).$(openssl rand -hex 12)"
```

Create a Kubernetes secret for the access key:

```
kubectl create secret generic infra-admin-access-key --from-literal=access-key=$INFRA_ACCESS_KEY
```

Configure an initial `values.yaml` file to use this admin access key:

```yaml
config:
  admin:
    enable: true
    accessKeySecret: infra-admin-access-key

server:
  service:
    type: LoadBalancer
```

Then deploy the Infra server:

```
helm install infra-server infrahq/infra-server -f values.yaml
```

Retrieve your Infra server endpoint:

```
export INFRA_SERVER_HOST=$(kubectl get service infra-server -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}")
```

Finally, use this access key in Terraform to set up `providers` and `grants`:

```hcl
terraform {
  required_providers {
    infra = {
      source = "infrahq/infra"
    }
  }
}

# Configure Infra Terraform provider.
provider "infra" {
  access_key = "$INFRA_ACCESS_KEY"
  host = "$INFRA_SERVER_HOST"
}

resource "infra_identity_provider" "okta" {
  client_id     = "0oa2hl2inow5Uqc6c357"
  client_secret = "lj1aj801208sdjf19820d122jhaljksdamkj0"

  okta {
    issuer = "example.okta.com"
  }
}

resource "infra_grant" "kubernetes_admin" {
  user_name = "example@example.com"

  kubernetes {
    cluster = "my_cluster"
    role    = "admin"
  }
}
```

And then run `terraform plan` or `terraform apply`:

```
terraform apply
```

#### Cleanup the initial admin (optional)

To disable the admin user in the Helm chart, update your values file:

```yaml
config:
  admin:
    enabled: false
```

Remove the admin access key:

```
kubectl delete secret infra-admin-access-key
```

Lastly, delete the generated admin user from Infra:

```
infra users remove admin@local
```

## PostgreSQL Database

Infra uses PostgreSQL as a data store, and the Infra server helm chart includes a PostgreSQL deployment. If
you use an external database please be aware of the following.

Infra is tested to work with the latest patch release of PostgreSQL 12.x and PostgreSQL 14.x.
Other versions may work, but are not tested.

Infra requires a dedicated database connection for every Infra connector, plus a few more
connections to handle other requests. It is important to ensure the postgres database
[`max_connections`](https://www.postgresql.org/docs/current/runtime-config-connection.html#GUC-MAX-CONNECTIONS)
is set accordingly, and also that the number of open connections is monitored.

## Customization

### Helm values

Refer to the [Helm chart](https://github.com/infrahq/helm-charts/tree/main/charts/infra-server) documentation for a full list of customization options.

### Load Balancer

To expose the Infra server externally via a load balancer service:

```yaml
# example values.yaml
---
server:
  service:
    type: LoadBalancer
```

### Internal Load Balancer

```yaml
# example values.yaml
---
server:
  service:
    annotations:
      # If using Google GKE
      cloud.google.com/load-balancer-type: Internal

      # If using AWS EKS
      service.beta.kubernetes.io/aws-load-balancer-scheme: internal

      # If using Azure AKS
      service.beta.kubernetes.io/azure-load-balancer-internal: true
```

### Health Check

```yaml
# example values.yaml
---
server:
  service:
    annotations:
      # If using AWS EKS
      service.beta.kubernetes.io/aws-load-balancer-healthcheck-protocol: HTTPS
      service.beta.kubernetes.io/aws-load-balancer-healthcheck-path: /healthz

      # If using Azure AKS
      service.beta.kubernetes.io/azure-load-balancer-health-probe-protocol: https # Kubernetes 1.20+
      service.beta.kubernetes.io/azure-load-balancer-health-probe-request-path: healthz # Kubernetes 1.20+

      # If using Digital Ocean
      service.beta.kubernetes.io/do-loadbalancer-healthcheck-protocol: https
      service.beta.kubernetes.io/do-loadbalancer-healthcheck-path: /healthz
```

### Ingress

Infra server can be configured to expose port 80 (HTTP) and 443 (HTTPS). Use the following Ingress controller specific examples to configure Infra Server Ingress.

> Infra connectors make long polling requests to the infra server. These requests may sit idle for up to 5 minutes while waiting for updates. The ingress must be configured with a timeout of at least 305 seconds to avoid 504 timeout errors from the connector.

#### Ambassador (Service Annotations)

```yaml
# example values.yaml
---
server:
  service:
    type: ClusterIP
    annotations:
      getambassador.io/config: |-
        apiVersion: getambassador.io/v2
        kind: Mapping
        name: infra-https-mapping
        namespace: infrahq
        host: infrahq.example.com                 # edit me
        prefix: /
        service: http://infra
        timeout_ms: 305000
        idle_timeout_ms: 305000
```

#### AWS Application Load Balancer Controller (ALB)

```yaml
# example values.yaml
---
server:
  ingress:
    enabled: true
    hosts:
      - infra.example.com # edit me
    className: alb
    paths:
      - '/*'
    annotations:
      alb.ingress.kubernetes.io/scheme: internet-facing # (optional: use "internal" for non-internet facing)
      alb.ingress.kubernetes.io/backend-protocol: HTTP
      alb.ingress.kubernetes.io/actions.ssl-redirect: '{"Type": "redirect", "RedirectConfig": { "Protocol": "HTTPS", "Port": "443", "StatusCode": "HTTP_301"}}'
      alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS":443}]'
      alb.ingress.kubernetes.io/target-type: ip
      alb.ingress.kubernetes.io/group.name: infra # (optional: edit me to use an existing shared load balanacer)
```

#### NGINX Ingress Controller

```yaml
# example values.yaml
---
server:
  ingress:
    enabled: true
    hosts:
      - infra.example.com # edit me
    servicePort: 80
    className: nginx
    annotations:
      nginx.ingress.kubernetes.io/force-ssl-redirect: 'true'
      nginx.ingress.kubernetes.io/backend-protocol: 'HTTP'
      nginx.ingress.kubernetes.io/proxy-http-version: '1.0'
      cert-manager.io/issuer: 'letsencrypt-prod' # edit me
      nginx.ingress.kubernetes.io/proxy-send-timeout: '305'
      nginx.ingress.kubernetes.io/proxy-read-timeout: '305'
    tls:
      - hosts:
          - infra.example.com # edit me
        secretName: com-example-infra # edit me
```

### Service Accounts

```yaml
# example values.yaml
---
server:
  serviceAccount:
    annotations:
      # Google Workload Identity
      # https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
      iam.gke.io/gcp-service-account: <GSA_NAME>@<PROJECT_ID>.iam.gserviceaccount.com

      # AWS Service Account Role
      # https://docs.aws.amazon.com/eks/latest/userguide/specify-service-account-role.html
      eks.amazonaws.com/role-arn: arn:aws:iam::<ACCOUNT_ID>:role/<IAM_ROLE_NAME>
```

## Uninstalling

```shell
helm uninstall infra-server
```

### Pinning a CLI version from Homebrew

To pin a specific version from Infra's [homebrew tap](https://github.com/infrahq/homebrew-tap/), use `brew pin`:

```bash
# Get the infra.rb file from the homebrew tap repo (e.g. here for 0.15.2)
curl https://raw.githubusercontent.com/infrahq/homebrew-tap/286489abe6266d3af9d19e364bacf9b960f8e696/infra.rb -o infra.rb

# install `infra` from this version
brew install infra.rb

# pin the version for now (i.e. don't auto upgrade)
brew pin infra
```
