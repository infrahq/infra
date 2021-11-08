# Infra Helm Chart

The Infra Helm chart is the recommended way of installing Infra on Kubernetes.

## Add Helm Repo

```bash
helm repo add infrahq https://helm.infrahq.com
helm repo update
```

## Install Infra

```bash
helm install -n infrahq --create-namespace infra infrahq/infra
```

## Advanced Service Account Configuration

```yaml
# example values.yaml
---
serviceAccount:
  annotations:
    # Google Workload Identity
    # https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
    iam.gke.io/gcp-service-account: <GSA_NAME>@<PROJECT_ID>.iam.gserviceaccount.com

    # AWS Service Account Role
    # https://docs.aws.amazon.com/eks/latest/userguide/specify-service-account-role.html
    eks.amazonaws.com/role-arn: arn:aws:iam::<ACCOUNT_ID>:role/<IAM_ROLE_NAME>
```

## Advanced Service Configuration

### Internal Load Balancer

```yaml
# example values.yaml
---
service:
  annotations:
    # Google GKE
    cloud.google.com/load-balancer-type: Internal

    # AWS EKS
    service.beta.kubernetes.io/aws-load-balancer-scheme: internal

    # Azure AKS
    service.beta.kubernetes.io/azure-load-balancer-internal: true
```

### Health Check

```yaml
# example values.yaml
---
service:
  annotations:
    # AWS EKS
    service.beta.kubernetes.io/aws-load-balancer-healthcheck-protocol: HTTPS
    service.beta.kubernetes.io/aws-load-balancer-healthcheck-path: /healthz

    # Azure AKS
    service.beta.kubernetes.io/azure-load-balancer-health-probe-protocol: https        # Kubernetes 1.20+
    service.beta.kubernetes.io/azure-load-balancer-health-probe-request-path: healthz  # Kubernetes 1.20+

    # Digital Ocean
    service.beta.kubernetes.io/do-loadbalancer-healthcheck-protocol: http
    service.beta.kubernetes.io/do-loadbalancer-healthcheck-path: /healthz
```

## Advanced Ingress Configuration

### Ambassador (Service Annotations)

```yaml
# example values.yaml
---
service:
  type: ClusterIP
  annotations:
    getambassador.io/config: |-
      apiVersion: getambassador.io/v2
      kind: Mapping
      name: infra-https-mapping
      namespace: {{ .Release.Namespace }}
      host: infrahq.example.com                 # edit me
      prefix: /
      service: http://infra
```

### AWS Application Load Balancer Controller (ALB)

```yaml
# example values.yaml
---
service:
  type: ClusterIP

ingress:
  enabled: true
  hosts: ["infra.example.com"]  # edit me
  annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internet-facing         # (optional: use "internal" for non-internet facing)
    alb.ingress.kubernetes.io/backend-protocol: HTTP
    alb.ingress.kubernetes.io/actions.ssl-redirect: '{"Type": "redirect", "RedirectConfig": { "Protocol": "HTTPS", "Port": "443", "StatusCode": "HTTP_301"}}'
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS":443}]'
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/group.name: infra               # (optional: edit me to use an existing shared load balanacer)
```

### NGINX Ingress Controller

```yaml
# example values.yaml
---
service:
  type: ClusterIP

ingress:
  enabled: true
  hosts: ["infra.example.com"]  # edit me
  servicePort: 80
    annotations:
      kubernetes.io/ingress.class: "nginx"
      nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
      nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
      nginx.ingress.kubernetes.io/proxy-http-version: "1.0"
      cert-manager.io/issuer: "letsencrypt-prod"
```

## Uninstall Infra

```bash
# Remove Infra
helm uninstall -n infrahq infra

# Remove potential secrets created for Infra
kubectl delete -n infrahq secret/infra-okta
```

## Uninstall Infra Engine

```bash
# Remove Infra Engine
helm uninstall -n infrahq infra-engine

# Remove rolebindings & clusterrolebindings created by Infra Engine
kubectl delete clusterrolebindings,rolebindings -l app.kubernetes.io/managed-by=infra --all-namespaces
```

## Configuration Reference

### Infra

| Parameter                          | Description                             | Default                      |
|------------------------------------|-----------------------------------------|------------------------------|
| `config`                           | Infra configuration file                | `nil`                        |
| `crashReporting.enabled`           | Enable crash report                     | `true`                       |
| `engine.*`                         | Engine chart values                     | (see below)                  |
| `engine.apiKey`                    | Engine API key                          | `""` (generated)             |
| `telemetry.enabled`                | Enable telemetry collection             | `true`                       |
| `image.tag`                        | Image tag                               | `""` (latest release)        |
| `image.repository`                 | Image repository                        | `infrahq/infra`              |
| `image.pullPolicy`                 | Image pull policy                       | `IfNotPresent`               |
| `ingress.enabled`                  | Enable ingress                          | `false`                      |
| `ingress.host`                     | Ingress host                            | `""`                         |
| `ingress.tls`                      | Ingress tls configuration               | `[]`                         |
| `ingress.servicePort`              | Target http service port backend        | `80`                         |
| `ingress.annotations`              | Ingress annotations (https)             | `{}`                         |
| `ingress.labels`                   | Ingress labels (https)                  | `{}`                         |
| `pod.annotations`                  | Pod annotations                         | `{}`                         |
| `service.type`                     | Service type                            | `LoadBalancer`               |
| `service.ports`                    | Service ports                           | `[{name:http, port:80 targetPort:80}, {name:https, port:443, targetPort:43}]` |
| `service.ports[].port`             | Service port                            | `nil`                        |
| `service.ports[].name`             | Service port name                       | `nil`                        |
| `service.ports[].targetPort`       | Service port target                     | `nil`                        |
| `service.ports[].protocol`         | Service port protocol                   | `TCP`                        |
| `service.labels`                   | Service labels                          | `{}`                         |
| `service.annotations`              | Service annotations                     | `{}`                         |
| `service.loadBalancerIP`           | IP address to assign to load balancer   | `nil`                        |
| `service.loadBalancerSourceRanges` | List of IP CIDRs allowed access         | `[]`                         |
| `service.externalIPs`              | Service external IP addresses           | `[]`                         |
| `service.clusterIP`                | Internal cluster service IP             | `nil`                        |
| `serviceAccount.create`            | Enable service account creation         | `true`                       |
| `serviceAccount.annotations`       | Service account annotations             | `{}`                         |
| `storage`                          | Storage size                            | `1Gi`                        |

### Infra Engine

| Parameter                          | Description                             | Default                      |
|------------------------------------|-----------------------------------------|------------------------------|
| `name`                             | Cluster name                            | `""` (auto-discovered)       |
| `host`                             | Infra endpoint                          | `""` (required)              |
| `apiKey`                           | Infra API key                           | `""` (required)              |
| `labels`                           | Additional filter labels                | `[]`                         |
| `image.tag`                        | Image tag                               | `""` (latest release)        |
| `image.repository`                 | Image repository                        | `infrahq/infra`              |
| `image.pullPolicy`                 | Image pull policy                       | `IfNotPresent`               |
| `pod.annotations`                  | Pod annotations                         | `{}`                         |
| `service.type`                     | Service type                            | `LoadBalancer`               |
| `service.ports`                    | Service ports                           | `[{name:https, port:443, targetPort:43}]` |
| `service.ports[].port`             | Service port                            | `nil`                        |
| `service.ports[].name`             | Service port name                       | `nil`                        |
| `service.ports[].targetPort`       | Service port target                     | `nil`                        |
| `service.ports[].protocol`         | Service port protocol                   | `TCP`                        |
| `service.labels`                   | Service labels                          | `{}`                         |
| `service.annotations`              | Service annotations                     | `{}`                         |
| `service.loadBalancerIP`           | IP address to assign to load balancer   | `nil`                        |
| `service.loadBalancerSourceRanges` | List of IP CIDRs allowed access         | `[]`                         |
| `service.externalIPs`              | Service external IP addresses           | `[]`                         |
| `service.clusterIP`                | Internal cluster service IP             | `nil`                        |
| `serviceAccount.create`            | Enable service account creation         | `true`                       |
| `serviceAccount.annotations`       | Service account annotations             | `{}`                         |
