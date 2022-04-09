# Granting Access

## Grant access

To grant access, use `infra grants add`:

```
infra grants add user@example.com kubernetes.staging --role edit
```

Note: the same command can be used to grant access to a group, for example:

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
  PROVIDER  IDENTITY     ACCESS  RESOURCE                   
  okta      Everyone     edit    kubernetes.development
  okta      Engineering  edit    kubernetes.development.monitoring  
  okta      Design       edit    kubernetes.development.web 
  okta      Engineering  view    kubernetes.production
  okta      Engineering  edit    kubernetes.production.web
```
