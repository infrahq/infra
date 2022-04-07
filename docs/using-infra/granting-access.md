# Granting Access

## Grant access

To grant access, use `infra grants add`:

```
infra grants add --user user@example.com kubernetes.staging --role edit
```

Note: the same command can be used to grant access to a group, for eaxmple:

```
infra grants add --group engineering kubernetes.staging --role edit
```

## Revoking access

Access is revoked via `infra grants remove`:

```
infra grants remove --user user@example.com kubernetes.staging --role edit
```

## Viewing access

```
infra grants list
  PROVIDER  IDENTITY     ACCESS  RESOURCE                                 
  okta      Everyone     view    kubernetes.development-72f9584e          
  okta      Engineering  edit    kubernetes.development-72f9584e.infrahq  
  okta      Design       edit    kubernetes.development-72f9584e.web 
```

