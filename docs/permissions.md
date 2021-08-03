# Configure Roles

In Infra, roles are configured via the [configuration file](./configuration.md).

## Create a configuration file

First, create or edit an existing a config file `infra.yaml`:

```
users:
  - name: admin@example.com   # user email
    roles:
      - name: admin           # Kubernetes cluster role name
        kind: cluster-role
        clusters:             # clusters for which to apply
          - cluster-AAA
          - cluster-BBB
```

Then, apply it to the Infra registry:

```
helm upgrade infra --set-file config=./infra.yaml --recreate-pods
```
