---
title: Helm Reference
position: 3
---

# Helm Chart Reference

The Infra Helm chart is the recommended way of installing Infra on Kubernetes.

## Add the chart repository

```
helm repo add infrahq https://helm.infrahq.com
helm repo update
```

## Installing

```
helm upgrade --install infra infrahq/infra
```

## Customizing

To customize your Infra install, create a `values.yaml` file:

```yaml
# example values.yaml
server:
  replicas: 1
```

Then deploy Infra with these custom values:

```
helm upgrade --install infra infrahq/infra -f values.yaml
```

## Adding users, grants and providers

Users, grants and providers can be specified in code via the Helm chart:

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: Okta
        url: example.okta.com
        clientID: example_jsldf08j23d081j2d12sd
        clientSecret: example_plain_secret # see `secrets` below

    # Add an admin user
    users:
      - name: admin
        password: password

    grants:
      # 1. Grant user(s) or group(s) as Infra administrator
      # Setup an user as Infra administrator
      - user: admin
        role: admin
        resource: infra

      # 2. Grant user(s) or group(s) access to a resources
      # Example of granting access to an individual user the `cluster-admin` role. The name of a resource is specified when installing the Infra Engine at that location.
      - user: admin
        role: cluster-admin                  # cluster_roles required
        resource: example-cluster            # limit access to the `example-cluster` Kubernetes cluster

      # Example of granting access to an individual user through assigning them to the 'edit' role in the `web` namespace.
      # In this case, Infra will automatically scope the access to a namespace.
      - user: admin
        role: edit                            # cluster_roles required
        resource: example-cluster.web         # limit access to only the `web` namespace in the `example-cluster` Kubernetes cluster

      # Example of granting access to a group the `view` role.
      - group: Everyone
        role: view                           # cluster_roles required
        resource: example-cluster            # limit access to the `example-cluster` Kubernetes cluster
```

## Postgres Database

Infra can be configured to use Postgres as a data store:

```yaml
# example values.yaml
---
server:
  envFrom:
    - secretRef:
        name: my-postgres-secret

  config:
    dbHost: example.com
    dbPort: 5432
    dbName: myinfra
    dbUsername: myuser
    dbPassword: env:POSTGRES_DB_PASSWORD # populated from my-postgres-secret environment
```


## Services

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
      service.beta.kubernetes.io/azure-load-balancer-health-probe-protocol: https        # Kubernetes 1.20+
      service.beta.kubernetes.io/azure-load-balancer-health-probe-request-path: healthz  # Kubernetes 1.20+

      # If using Digital Ocean
      service.beta.kubernetes.io/do-loadbalancer-healthcheck-protocol: https
      service.beta.kubernetes.io/do-loadbalancer-healthcheck-path: /healthz
```

## Ingress

Infra server can be configured exposes port 80 (HTTP) and 443 (HTTPS). Use the following Ingress controller specific examples to configure Infra server Ingress.

### Ambassador (Service Annotations)

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
```

### AWS Application Load Balancer Controller (ALB)

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
server:
  ingress:
    enabled: true
    hosts:
      - infra.example.com # edit me
    servicePort: 80
    className: nginx
    annotations:
      nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
      nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
      nginx.ingress.kubernetes.io/proxy-http-version: "1.0"
      cert-manager.io/issuer: "letsencrypt-prod" # edit me
    tls:
      - hosts:
          - infra.example.com          # edit me
        secretName: com-example-infra  # edit me
```

## Secrets

Infra supports many secret storage backends, including, but not limited to:

- Kubernetes
- Vault
- AWS Secrets Manager
- AWS SSM (Systems Manager Parameter Store)
- Environment variables
- Files on the file system
- plaintext secrets (though not recommended)

### Kubernetes

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: kubernetes:infra-okta/clientSecret
```

This would read the `infra-okta/clientSecret` key from a Kubernetes secret.

Kubernetes takes configuration, like so:

```yaml
# example values.yaml
---
server:
  config:
    secrets:
      - name: kubernetes # can optionally provide a custom name
        kind: kubernetes
        config:
          namespace: mynamespace
```

namespace defaults to whatever is defined in `/var/run/secrets/kubernetes.io/serviceaccount/namespace`, or the `default` namespace.

### Vault

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: vault:infra-okta-clientSecret
```

This would read the `infra-okta-clientSecret` secret from Vault

Vault takes configuration, like so:

```yaml
server:
  config:
    secrets:
      - name: vault # can optionally provide a custom name
        kind: vault
        config:
          transitMount: /transit
          secretMount: /secret
          token: env:VAULT_TOKEN # secret config can even reference other built-in secret types, like env
          namespace: mynamespace
          address: https://vault
```

### AWS Secrets Manager

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: awssm:infra-okta-clientSecret
```

Secrets Manager takes configuration, like so:

```yaml
# example values.yaml
---
server:
  config:
    secrets:
      - name: awssm # can optionally provide a custom name
        kind: awssecretsmanager
        config:
          endpoint: https://kms.endpoint
          region: us-west-2
          accessKeyID: env:AWS_ACCESS_KEY_ID # secret config can even reference other built-in secret types, like env
          secretAccessKey: env:AWS_SECRET_ACCESS_KEY
```

### AWS SSM (Systems Manager Parameter Store)

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: awsssm:infra-okta-clientSecret
```

SSM takes configuration, like so:

```yaml
# example values.yaml
---
server:
  config:
    secrets:
      - name: awsssm # can optionally provide a custom name
        kind: awsssm
        config:
          keyID: 1234abcd-12ab-34cd-56ef-1234567890ab # optional, if set it's the KMS key that should be used for decryption
          endpoint: https://kms.endpoint
          region: us-west-2
          accessKeyID: env:AWS_ACCESS_KEY_ID # secret config can even reference other built-in secret types, like env
          secretAccessKey: env:AWS_SECRET_ACCESS_KEY
```

### Environment variables

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: env:OKTA_CLIENT_SECRET
```

**env** is built-in and does not need to be declared, but if you do want to declare the configuration for it, you could use this to create a custom env handler which base64 encodes the secret:

```yaml
# example values.yaml
---
server:
  config:
    secrets:
      - name: base64env
        kind: env
        config:
          base64: true
          base64UrlEncoded: false
          base64Raw: false
```

which you would then use like this. First define an environment variable in the context where it will be accessed. [There are many ways to do this in Kubernetes](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/). Typically the environment variable in Kubernetes will be defined in the [deployment](/helm/charts/infra/templates/server/deployment.yaml). To temporarily define an environment variable you can use `kubectl`:

```bash
$ kubectl set env deployment/infra OKTA_CLIENT_SECRET=c3VwZXIgc2VjcmV0IQ==
```

Then use the name of the secret back-end and the name of the environment variable in the `infra.yaml` file:

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: base64env:OKTA_CLIENT_SECRET
```

### Files on the file system

It's a common pattern to write secrets to a set of files on disk and then have an app read them. Note that one secret is stored per file in plaintext.

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: file:/var/secrets/okta-client-secret.txt
```

**file** is built-in and does not need to be declared, but if you do want to declare the configuration for it, you could use this to create a custom handler, like so:

```yaml
# example values.yaml
---
server:
  config:
    secrets:
      - name: base64file
        kind: file
        config:
          base64: true
          base64UrlEncoded: false
          base64Raw: false
          path: /var/secrets # optional: assume all files mentioned are in this root directory
```

which you would then use as follows. First base64 encode a string and write it to a file:

```bash
$ echo "c3VwZXIgc2VjcmV0IQ==" > /var/secrets/okta-client-secret.txt
```

Then in the `infra.yaml` file, use the name of the secrets config declaration and then the name of the file.

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: base64file:okta-client-secret.txt
```

### plaintext secrets (though probably not recommended)

Sometimes it can be handy to support plain text secrets right in the yaml config, especially when the yaml is being generated and the secrets are coming from elsewhere.

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: plaintext:mySupErSecrEt
```

Optionally for plaintext secrets, you can leave off the secret back-end name:

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: mySupErSecrEt
```

**plaintext** is built-in and does not need to be declared, but if you do want to declare the configuration for it so that you can include base64 encoded strings, you could use this to create a custom handler:

```yaml
# example values.yaml
---
server:
  config:
    secrets:
      - name: base64text
        kind: plain
        config:
          base64: true
          base64UrlEncoded: false
          base64Raw: false
```

Which you would then use in the `infra.yaml` file as shown:

```yaml
# example values.yaml
---
server:
  config:
    providers:
      - name: okta
        clientSecret: base64text:bXlTdXBFclNlY3JFdA==
```

## Encryption

### Encryption Keys

Sensitive data is always encrypted at rest in the db using a symmetric key. The symmetric key is stored in the database encrypted by a root key. By default this root key is generated by Infra and stored in a secret (default: `~/.infra/key`, or in Kubernetes, a as secret named `infra-x` with the key `/__root_key`). Encrpytion at rest can be configured using another key provider service such as KMS or Vault.

The process of retrieving the db key is to load the encrypted key from the database, request that the db key be decrypted by the root key, and at which point the db key is used to decrypt all the data. In the case of AWS KMS and Vault, the Infra app never sees the root key, and so these options are preferred over the default built-in `native` key provider.

### Root key configuration examples

Infra uses AWS KMS key service:

```yaml
# example values.yaml
---
server:
  config:
    keys:
      - kind: awskms
        endpoint: https://your.kms.aws.url.example.com
        region: us-east-1
        accessKeyId: kubernetes:awskms/accessKeyID
        secretAccessKey: kubernetes:awskms/secretAccessKey
        encryptionAlgorithm: AES_256
```

Infra uses Vault as a key service:

```yaml
# example values.yaml
---
server:
  config:
    keys:
      - kind: vault
        address: https://your.vault.url.example.com
        transitMount: /transit
        token: kubernetes:vault/token
        namespace: namespace
```

By default, Infra will manage keys internally. You can use a predefined 256-bit cryptographically random key by creating and mounting a secret to the server pod.

```yaml
# example values.yaml
---
server:
  volumes:
    - name: my-db-encryption-secret
      secret:
        secretName: my-db-encryption-secret
  volumeMounts:
    - name: my-encryption-secret
      mountPath: /var/run/secrets/my/db/encryption/secret

  config:
    dbEncryptionKey: /var/run/secret/my/db/encryption/secret
```

If an encryption key is not provided, one will be randomly generated during install time. It is the responsibility of the operator to back up this key.

## Service Accounts

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

```bash
# Remove Infra
helm uninstall infra

# Remove rolebindings & clusterrolebindings created by Infra connector
kubectl delete clusterrolebindings,rolebindings -l app.kubernetes.io/managed-by=infra --all-namespaces
```
