# Using an External Postgres Database with Infra

When using Infra in a production environment configuring an external Postgres database is recommended. This database should be backed up on a regular interval.

## Configuration

```yaml
# example values.yaml
---
server:
  envFrom:
    - secretRef:
        name: my-infrahq-secrets

  config:
    dbHost: example.com
    dbPort: 5432
    dbName: myinfra
    dbUser: myuser
    dbPassword: env:POSTGRES_DB_PASSWORD # the password can be populated from my-infrahq-secrets injected into the environment
```
