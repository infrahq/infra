---
title: Helm Reference
position: 3
---

# Helm Chart Reference

The Infra Helm chart is the recommended way of installing Infra on Kubernetes.

## Add the chart repository

```bash
helm repo add infrahq https://helm.infrahq.com
helm repo update
```

## Installing

```bash
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

```bash
helm upgrade --install infra infrahq/infra -f values.yaml
```

## Users

Add users to the Helm values file by specifying a name which should be a valid email address, and a password or access key. The password is the more likely way to validate the user, though a process or service account will probably use an access key. Access keys must be in the form of XXXXXXXXXX.YYYYYYYYYYYYYYYYYYYYYYYY (<10 character ascii key>.<24 character ascii secret>). Here are three of the more common ways of dealing with secrets. For more options, scroll down to the [Secrets](#secrets) section.

```yaml
server:
  config:
    users:
      # Add a user with a plaintext password
      - name: admin@example.com
        password: SetThisPassword! 

      # Add a user with a plaintext access key.
      - name: admin@example.com
        accessKey: 123bogusab.abcdefnotreal123key456ab

      # Add a user setting the password using a file. The password should be the only contents of the file. The 
      # file will need to be mounted into the pod using `volumes` and `volumeMounts`.
      - name: admin@example.com
        password: file:/var/run/secrets/admin@example.com

      # Add a user setting the access key using a file. The access key should be the only contents of the file. The 
      # file will need to be mounted into the pod using `volumes` and `volumeMounts`.
      - name: admin@example.com
        accessKey: file:/var/run/secrets/admin@example.com

      # add a user setting the password with an environment variable. The environment variable will need to be 
      # injected into the pod using `env` or `envFrom`.
      - name: admin@example.com
        password: env:ADMIN_PASSWORD

      # add a user setting the access key with an environment variable. The environment variable will need to be 
      # injected into the pod using `env` or `envFrom`.
      - name: admin@example.com
        accessKey: env:ACCESS_KEY
```

## Grants

For each user and resource (Infra or a Kubernetes cluster) defined, you can add a grant. A grant includes a user name or group name, a role, and a resource.

```yaml
# example values.yaml
---
server:
  config:
    grants:
      # 1. Grant user(s) or group(s) as Infra administrator
      # Setup a user as Infra administrator
      - user: admin@example.com
        role: admin
        resource: infra

      # 2. Grant user(s) or group(s) access to a resources
      # Example of granting access to an individual user the `cluster-admin` role. The name of a resource is specified when installing the Infra Engine at that location.
      - user: admin@example.com
        role: cluster-admin # Roles for Kubernetes clusters must be Cluster Roles. 
        resource: example-cluster # limit access to the `example-cluster` Kubernetes cluster

      # Example of granting access to an individual user through assigning them to the 'edit' role in the `web` namespace.
      # In this case, Infra will automatically scope the access to a namespace.
      - user: admin@example.com
        role: edit # cluster_roles required
        resource: example-cluster.web # limit access to only the `web` namespace in the `example-cluster` Kubernetes cluster

      # Example of granting access to a group the `view` role.
      - group: Everyone
        role: view # cluster_roles required
        resource: example-cluster # limit access to the `example-cluster` Kubernetes cluster
```

## OIDC Providers

OIDC Providers, such as Okta, Azure AD, Google, and others can be added using the Client ID, Secret, Name, and URL.

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
      service.beta.kubernetes.io/azure-load-balancer-health-probe-protocol: https # Kubernetes 1.20+
      service.beta.kubernetes.io/azure-load-balancer-health-probe-request-path: healthz # Kubernetes 1.20+

      # If using Digital Ocean
      service.beta.kubernetes.io/do-loadbalancer-healthcheck-protocol: https
      service.beta.kubernetes.io/do-loadbalancer-healthcheck-path: /healthz
```

## Ingress

Infra server can be configured exposes port 80 (HTTP) and 443 (HTTPS). Use the following Ingress controller specific examples to configure Infra Server Ingress.

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
      - "/*"
    annotations:
      alb.ingress.kubernetes.io/scheme: internet-facing # (optional: use "internal" for non-internet facing)
      alb.ingress.kubernetes.io/backend-protocol: HTTP
      alb.ingress.kubernetes.io/actions.ssl-redirect: '{"Type": "redirect", "RedirectConfig": { "Protocol": "HTTPS", "Port": "443", "StatusCode": "HTTP_301"}}'
      alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS":443}]'
      alb.ingress.kubernetes.io/target-type: ip
      alb.ingress.kubernetes.io/group.name: infra # (optional: edit me to use an existing shared load balanacer)
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
          - infra.example.com # edit me
        secretName: com-example-infra # edit me
```

## Secrets

Infra supports many secret storage backends, including, but not limited to:

- Kubernetes
- Vault
- AWS Secrets Manager
- AWS SSM (Systems Manager Parameter Store)
- Environment variables
- Files on the filesystem
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

**env** is built-in and does not need to be declared. If you do want to declare the configuration for the **env**, you could use this to create a custom env handler which base64 encodes the secret:

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

To use this, first define an environment variable in the context where it will be accessed. [There are many ways to do this in Kubernetes](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/). Typically, the environment variable in Kubernetes will be defined in the [deployment](/helm/charts/infra/templates/server/deployment.yaml). To temporarily define an environment variable you can use `kubectl`:

```bash
kubectl set env deployment/infra OKTA_CLIENT_SECRET=c3VwZXIgc2VjcmV0IQ==
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

It's a common pattern to write secrets to a set of files on disk and then have an app read them. Note that each file can store a single secret, and that secret must be in plaintext.

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

To use the file, first base64 encode a string and write it to a file:

```bash
echo "c3VwZXIgc2VjcmV0IQ==" > /var/secrets/okta-client-secret.txt
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

Sometimes it can be handy to support plain text secrets right in the YAML config, especially when the YAML is being generated and the secrets are coming from elsewhere.

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

You can then use this in the `infra.yaml` file as shown:

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

The database always encrypts sensitive data at rest using a symmetric key. The symmetric key is stored in the database encrypted by a root key. By default, Infra generates this root key and stores it in a secret (default: `~/.infra/key`, or in Kubernetes, as a secret named `infra-x` with the key `/__root_key`). Encryption at rest can be configured using another key provider service such as KMS or Vault.

The process of retrieving the database key is to load the encrypted key from the database, request that the database key be decrypted by the root key, and at which point the database key is used to decrypt all the data. In the case of AWS KMS and Vault, the Infra app never sees the root key, and so these options are preferred over the default built-in `native` key provider.

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

If an encryption key is not available, one will be generated during install time. It is the responsibility of the operator to back up this key.

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
