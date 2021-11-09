# Using an External Postgres Database with Infra

## Configuration
Postgres connections are made using data source connection name strings. These strings should follow a structure similar to the following format:
```bash
host=${POSTGRES_HOST} user=${POSTGRES_USER} password=${POSTGRES_PASSWORD} dbname=${POSTGRES_DB_NAME} port=${POSTGRES_PORT} sslmode=${POSTGRES_SSL_MODE}
```
Set the `pgsql-dsn` value in your Infra config:
```yaml
# example values.yaml
---
pgsql-dsn: host=${POSTGRES_HOST} user=${POSTGRES_USER} password=${POSTGRES_PASSWORD} dbname=${POSTGRES_DB_NAME} port=${POSTGRES_PORT} sslmode=${POSTGRES_SSL_MODE}
image:
  tag: 0.0.0-development
  pullPolicy: Never

config:
  # etc...
```