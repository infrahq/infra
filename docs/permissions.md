# Configure Roles

In Infra, roles are configured via the [configuration file](./configuration.md).

## Create a configuration file

First, create or edit an existing a config file `infra.yaml`:

```
groups:
  - name: developers          # group name in an external identity provider
    source: okta              # the identity provider this group applies to
    roles:
      - name: writer          # Kubernetes cluster-role name
        kind: cluster-role
        destinations:
          - name: cluster-AAA # clusters for which to apply
            namespaces:       # optional namespaces
              - default
              - web

users:
  - name: admin@example.com   # user email
    groups:                   # manually assign groups this user belongs to
      - developers
    roles:
      - name: admin           # Kubernetes cluster-role name
        kind: cluster-role
        destinations:
          - name: cluster-AAA # clusters for which to apply
          - name: cluster-BBB
```

Then, apply it to the Infra registry:

```
helm upgrade infra-registry infrahq/registry --set-file config=./infra.yaml
```
