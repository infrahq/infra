# Granting Access

## Roles

Infra allows granting different levels of access via **roles**, such as `view`, `edit` or `admin`. Each connector has different roles that can be used:

- [Kubernetes Roles](../connectors/kubernetes.md#roles)

## Grant access

To grant access, use `infra grants add`:

```
infra grants add user@example.com kubernetes.staging --role edit
```

Note: the same command can be used to grant access to a group using the boolean [--group] flag, for example:

```
infra grants add --group engineering kubernetes.staging --role edit
```

## Revoking access

Access is revoked via `infra grants remove`:

```
infra grants remove user@example.com kubernetes.staging --role edit
```

## Viewing access

```
infra grants list
  PROVIDER  IDENTITY     ACCESS  DESTINATION                   
  okta      Everyone     edit    kubernetes.development
  okta      Engineering  edit    kubernetes.development.monitoring  
  okta      Design       edit    kubernetes.development.web 
  okta      Engineering  view    kubernetes.production
  okta      Engineering  edit    kubernetes.production.web
```
