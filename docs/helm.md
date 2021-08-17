# Infra Helm Chart

The Infra helm chart is the recommended way of installing Infra on Kubernetes.

## Add Repo

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
```

## Install the Infra Registry

```
helm install infra-registry infrahq/registry --namespace infrahq --create-namespace
```

## Advanced Load Balancer Configuration

### Internal Load Balancer

```
# example values.yaml

service:
  annotations:
    # Google GKE
    cloud.google.com/load-balancer-type: "Internal"

    # AWS EKS
    service.beta.kubernetes.io/aws-load-balancer-internal: "true"

    # Azure AKS
    service.beta.kubernetes.io/azure-load-balancer-internal: "true"
```

## Advanced Ingress Configuration

### Ambassador (Service Annotations)

```
# example values.yaml

service:
  type: ClusterIP
  annotations:
    getambassador.io/config: |-
      ---
      apiVersion: getambassador.io/v2
      kind: Mapping
      name: infra-grpc-mapping
      namespace: {{ .Release.Namespace }}
      host: infrahq.example.com:443             # edit me
      prefix: /
      grpc: True
      service: infra-registry:80
      ---
      apiVersion: getambassador.io/v2
      kind: Mapping
      name: infra-https-mapping
      namespace: {{ .Release.Namespace }}
      host: infrahq.example.com                 # edit me
      prefix: /
      service: http://infra-registry
```

### AWS Application Load Balancer Controller (ALB)

```
# example values.yaml

service:
  type: ClusterIP

ingress:
  enabled: true
  hosts: ["infra.example.com"]  # edit me
  https:
    annotations:
      kubernetes.io/ingress.class: alb
      alb.ingress.kubernetes.io/scheme: internet-facing         # (optional: use "internal" for non-internet facing)
      alb.ingress.kubernetes.io/backend-protocol: HTTP
      alb.ingress.kubernetes.io/actions.ssl-redirect: '{"Type": "redirect", "RedirectConfig": { "Protocol": "HTTPS", "Port": "443", "StatusCode": "HTTP_301"}}'
      alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS":443}]'
      alb.ingress.kubernetes.io/target-type: ip
      alb.ingress.kubernetes.io/group.name: infra-registry      # (optional: edit me to use an existing shared load balanacer)
  grpc:
    annotations:
      kubernetes.io/ingress.class: alb
      alb.ingress.kubernetes.io/scheme: internet-facing         # (optional: use "internal" for non-internet facing)
      alb.ingress.kubernetes.io/backend-protocol-version: GRPC
      alb.ingress.kubernetes.io/listen-ports: '[{"HTTPS":443}]'
      alb.ingress.kubernetes.io/target-type: ip
      alb.ingress.kubernetes.io/group.name: infra-registry      # (optional: edit me to use an existing shared load balanacer)
```

### `ingress-nginx`

```
# example values.yaml

service:
  type: ClusterIP

ingress:
  enabled: true
  hosts: ["infra.example.com"]  # edit me
  https:
    servicePort: 80
    annotations:
      kubernetes.io/ingress.class: "nginx"
      nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
      nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
      nginx.ingress.kubernetes.io/proxy-http-version: "1.0"
      cert-manager.io/issuer: "letsencrypt-prod"
  grpc:
    servicePort: 443
    annotations:
      kubernetes.io/ingress.class: "nginx"
      ingress.kubernetes.io/ssl-redirect: "true"
      nginx.ingress.kubernetes.io/backend-protocol: "GRPCS"
      cert-manager.io/issuer: "letsencrypt-prod"
```

## Configuration Reference

| Parameter                                 | Description                                   | Default                                                 |
|-------------------------------------------|-----------------------------------------------|---------------------------------------------------------|
| `image.repository`                        | Image repository                              | `infrahq/registry`                                      |
| `image.tag`                               | Image tag                                     | Most recent version of Infra                            | 
| `image.pullPolicy`                        | Image Pull Policy                             | `IfNotPresent`                                          |
| `service.type`                            | Service type                                  | `LoadBalancer`                                          |
| `service.port`                            | Port to expose the plaintext service on       | `80`                                                    |
| `service.targetPort`                      | Target plaintext container port               | `80`                                                    |
| `service.portName`                        | Name of the plaintext service port            | `plaintext`                                             |
| `service.nodePort`                        | Service plaintext nodeport                    | `nil`                                                   |
| `service.tlsPort`                         | Port to expose the TLS service on             | `443`                                                   |
| `service.tlsTargetPort`                   | Target TLS container port                     | `443`                                                   |
| `service.tlsPortName`                     | Name of the TLS service port                  | `tls`                                                   |
| `service.tlsNodePort`                     | Service TLS nodeport                          | `nil`                                                   |
| `service.annotations`                     | Service annotations                           | `{}`                                                    |
| `service.labels`                          | Service labels                                | `{}`                                                    |
| `service.loadBalancerIP`                  | IP address to assign to load balancer         | `nil`                                                   |
| `service.loadBalancerSourceRanges`        | List of IP CIDRs allowed access               | `[]`                                                    |
| `service.externalIPs`                     | Service external IP addresses                 | `[]`                                                    |
| `service.clusterIP`                       | Internal cluster service IP                   | `nil`                                                   |
| `ingress.enabled`                         | Enable ingress                                | `false`                                                 |
| `ingress.host`                            | Ingress host                                  | `""`                                                    |
| `ingress.tls`                             | Ingress tls configuration                     | `[]`                                                    |
| `ingress.https.servicePort`               | Target http service port backend              | `80`                                                    |
| `ingress.https.annotations`               | Ingress annotations (https)                   | `{}`                                                    |
| `ingress.https.labels`                    | Ingress labels (https)                        | `{}`                                                    |
| `ingress.grpc.servicePort`                | Target grpc service port backend              | `80`                                                    |
| `ingress.grpc.annotations`                | Ingress annotations (grpc)                    | `{}`                                                    |
| `ingress.grpc.labels`                     | Ingress labels (grpc)                         | `{}`                                                    |