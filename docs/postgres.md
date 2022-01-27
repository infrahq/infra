# Using an External Postgres Database with Infra

When using Infra in a production environment configuring an external Postgres database is recommended. This database should be backed up on a regular interval.

## Configuration

```yaml
# example values.yaml
---
config:
  dbHost: example.com
  dbPort: 5432
  dbName: myinfra
  dbUser: myuser
  dbPassword: kubernetes:infra-postgres/pass # the password can be populated from Infra secrets, in this example a Kubernetes secret is used
# etc...
```