# Secrets

Infra supports many secret storage backends, including, but not limited to:

- Kubernetes
- Vault
- AWS Secrets Manager
- AWS SSM (Systems Manager Parameter Store)
- Environment variables
- Files on the file system
- plaintext secrets (though not recommended)

## Usage

These can be referenced in the Infra config file using the scheme `<secret-backend>:<secret-key>`

Examples follow.

### Kubernetes

```yaml
    clientSecret: kubernetes:infra-okta/clientSecret
```

This would read the `infra-okta/clientSecret` key from a Kubernetes secret.

Kubernetes takes configuration, like so:

```yaml
secrets:
  - name: kubernetes # can optionally provide a custom name
    kind: kubernetes
    namespace: mynamespace
```

### Vault

```yaml
    clientSecret: vault:infra-okta-clientSecret
```

This would read the `infra-okta-clientSecret` secret from Vault

Vault takes configuration, like so:

```yaml
secrets:
  - name: vault # can optionally provide a custom name
    kind: vault
    transitMount: /transit
    secretMount: /secret
    token: env:VAULT_TOKEN # secret config can even reference other built-in secret types, like env
    namespace: mynamespace
    address: https://vault
```

### AWS Secrets Manager

```yaml
    clientSecret: awssm:infra-okta-clientSecret
```

Secrets Manager takes configuration, like so:

```yaml
secrets:
  - name: awssm # can optionally provide a custom name
    kind: awssecretsmanager
    endpoint: https://kms.endpoint
    region: us-west-2
    accessKeyID: env:AWS_ACCESS_KEY_ID # secret config can even reference other built-in secret types, like env
    secretAccessKey: env:AWS_SECRET_ACCESS_KEY
```

### AWS SSM (Systems Manager Parameter Store)

```yaml
    clientSecret: awsssm:infra-okta-clientSecret
```

SSM takes configuration, like so:

```yaml
secrets:
  - name: awsssm # can optionally provide a custom name
    kind: awsssm
    keyID: 1234abcd-12ab-34cd-56ef-1234567890ab # optional, if set it's the KMS key that should be used for decryption
    endpoint: https://kms.endpoint
    region: us-west-2
    accessKeyID: env:AWS_ACCESS_KEY_ID # secret config can even reference other built-in secret types, like env
    secretAccessKey: env:AWS_SECRET_ACCESS_KEY
```

### Environment variables

```yaml
    clientSecret: env:OKTA_CLIENT_SECRET
```

**env** is built-in and does not need to be declared, but if you do want to declare the configuration for it, you could use this to create a custom env handler which base64 encodes the secret:

```yaml
secrets:
  - name: base64env
    kind: env
    base64: true
    base64UrlEncoded: false
    base64Raw: false
```

which you would then use like this. First create the environment variable:

```bash
$ export OKTA_CLIENT_SECRET="c3VwZXIgc2VjcmV0IQ=="
```

Then use the name of the secret backend and the name of the environment variable in the `infra.yaml` file:

```yaml
    clientSecret: base64env:OKTA_CLIENT_SECRET
```

### Files on the file system

It's a common pattern to write secrets to a set of files on disk and then have an app read them. Note that one secret is stored per file in plaintext.

```yaml
    clientSecret: file:/var/secrets/okta-client-secret.txt
```

**file** is built-in and does not need to be declared, but if you do want to declare the configuration for it, you could use this to create a custom handler which , like so:

```yaml
secrets:
  - name: base64file
    kind: file
    base64: true
    base64UrlEncoded: false
    base64Raw: false
    path: /var/secrets # optional: assume all files mentioned are in this root directory
```

which you would then use as follows. First base64 encode a string and write it to a file:

```bash
$ echo "c3VwZXIgc2VjcmV0IQ==" > /var/secrets/okta-client-secret.txt
```

then in the `infra.yaml` file, use the name of the secrets config declaration and then the name of the file. 

```yaml
    clientSecret: base64file:okta-client-secret.txt
```

### plaintext secrets (though probably not recommended)

Sometimes it can be handy to support plain text secrets right in the yaml config, especially when the yaml is being generated and the secrets are coming from elsewhere.

```yaml
    clientSecret: plaintext:mySupErSecrEt
```

Optionally for plaintext secrets, you can leave off the secret backend name for plaintext secrets:

```yaml
    clientSecret: mySupErSecrEt
```

**plaintext** is built-in and does not need to be declared, but if you do want to declare the configuration for it so that you can include base64 encoded strings, you could use this to create a custom handler:

```yaml
secrets:
  - name: base64text
    kind: plain
    base64: true
    base64UrlEncoded: false
    base64Raw: false
```

which you would then use in the `infra.yaml` file as shown:

```yaml
    clientSecret: base64text:bXlTdXBFclNlY3JFdA==
```
