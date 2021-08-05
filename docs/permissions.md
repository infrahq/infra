# Configure Roles

In Infra, roles are configured via the [configuration file](./configuration.md).

## Create a configuration file

First, create or edit an existing a config file `infra.yaml`:

```
groups:
  - name: developers          # group name in an external identity provider
    sources:                  # the identity providers this group applies to
      - okta
    roles:
      - name: writer          # Kubernetes cluster-role name
        kind: cluster-role
        clusters:             # clusters for which to apply
          - cluster-AAA

users:
  - name: admin@example.com   # user email
    groups:                   # manually assign groups this user belongs to
      - developers
    roles:
      - name: admin           # Kubernetes cluster-role name
        kind: cluster-role
        clusters:             # clusters for which to apply
          - cluster-AAA
          - cluster-BBB
```

Then, apply it to the Infra registry:

```
helm upgrade infra infrahq/infra --set-file config=./infra.yaml --recreate-pods
```
