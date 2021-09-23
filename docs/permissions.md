# Configure Roles

In Infra, roles are configured via the [configuration file](./configuration.md).

## Create a configuration file

First, create or edit an existing a config file `infra.yaml`:

```
groups:
  - name: developers          # group name in an external identity provider
    source: okta              # the identity provider this group applies to
    roles:
      - name: edit            # Kubernetes cluster-role name
        kind: cluster-role
        destinations:         # destinations for which this group-role mapping applies
          - name: cluster-AAA
            namespaces:       # optional namespaces
              - default
              - web

users:
  - email: admin@example.com   # user email
    source: okta
    groups:                   # manually assign groups this user belongs to
      - developers
    roles:
      - name: admin           # Kubernetes cluster-role name
        kind: cluster-role
        destinations:         # destinations for which this group-role mapping applies
          - name: cluster-AAA
          - name: cluster-BBB
```

Then, apply it to the Infra registry:

```
helm upgrade infra-registry infrahq/registry --set-file config=./infra.yaml
```
