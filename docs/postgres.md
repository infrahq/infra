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

### Connecting to Postgres by URL
Connecting to a Postgres database is also possible by explicitly specifying the URL:
```yaml
# example values.yaml
---
pg:
  url: postgres://myuser:{{kubernetes:infra-postgres/pass}}@example.com:5432/myinfra
image:
  tag: 0.0.0-development
  pullPolicy: Never

config:
  # etc...
```
The Postgres URL contains sensitive information such as the password to connect to the database. For this reason you may wish to use [secret providers](./docs/secrets) when specifying your URL. This can be done by wrapping the secret value in curly brackets (`{{ }}`) and specifying the secret value in the standard secret reference syntax.

For example, to retrieve a Postgres password from a Kubernetes secret:
```
{{kubernetes:infra-postgres/password}}
```