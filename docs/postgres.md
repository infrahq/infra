# Using an External Postgres Database with Infra

When using Infra in a production environment configuring an external Postgres database is recommended. This database should be backed up on a regular interval.

## Configuration
Postgres connections are made by populating the `pg` configuration in the Infra helm deployment:
```yaml
# example values.yaml
---
pg:
  host: example.com
  port: 5432
  db-name: myinfra
  user: myuser
  password: kubernetes:infra-postgres/pass # the password can be populated from Infra secrets, in this example a Kubernetes secret is used
image:
  tag: 0.0.0-development
  pullPolicy: Never

config:
  # etc...
```