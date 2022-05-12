# Granting Access

## Roles

Infra allows granting different levels of access via **roles**, such as `view`, `edit` or `admin`. Each connector has different roles that can be used:

- [Kubernetes Roles](../connectors/kubernetes.md#roles)

## Grant access

To grant access, use `infra grants add`. For example, to give `edit` access to a cluster named `staging` run:

```
infra grants add user@example.com staging --role edit
```

Note: the same command can be used to grant access to a group using the boolean [--group] flag, for example:

```
infra grants add --group engineering staging --role edit
```

## Revoking access

Access is revoked via `infra grants remove`:

```
infra grants remove user@example.com staging --role edit
```

## Viewing access

```
infra grants list
  USER                 ACCESS   DESTINATION
  jeff@infrahq.com     edit     development
  michael@infrahq.com  view     production

  GROUP          ACCESS    DESTINATION
  Engineering    edit      development.monitoring
  Engineering    view      production
  Design         edit      development.web
```
