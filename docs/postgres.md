# Using an External Postgres Database with Infra

When using Infra in a production environment configuring an external Postgres database is recommended. This database should be backed up at a regular interval.

## Configuration
Postgres connections are made using data source connection name (DSNs) strings or Postgres connection strings. These strings should follow a structure similar to the following format (where you can set the environment variables to suit your deployment):
```bash
host=${POSTGRES_HOST} user=${POSTGRES_USER} password=${POSTGRES_PASSWORD} dbname=${POSTGRES_DB_NAME} port=${POSTGRES_PORT} sslmode=${POSTGRES_SSL_MODE}
```
or
```
postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB_NAME}
```
To set the Infra database to your Postgres instance set the `postgres` value in your Infra config:
```yaml
# example values.yaml
---
postgres: host=${POSTGRES_HOST} user=${POSTGRES_USER} password=${POSTGRES_PASSWORD} dbname=${POSTGRES_DB_NAME} port=${POSTGRES_PORT} sslmode=${POSTGRES_SSL_MODE}
image:
  tag: 0.0.0-development
  pullPolicy: Never

config:
  # etc...
```

### Using Infra Secrets in Postgres Configuration
The Postgres data source connection name contains sensitive information such as the password to connect to the database. For this reason you may wish to use [secret providers](./docs/secrets) when specifying your DSN. This can be done by wrapping the secret value in curly brackets (`{{ }}`) and specifying the secret value in the standard secret reference syntax.

For example, to retrieve a Postgres password from a Kubernetes secret:
```
postgres: host=${POSTGRES_HOST} user=${POSTGRES_USER} password={{kubernetes:infra-postgres/password}} dbname=${POSTGRES_DB_NAME} port=${POSTGRES_PORT} sslmode=${POSTGRES_SSL_MODE}
```