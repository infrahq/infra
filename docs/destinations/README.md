# Destination

## Supported Destinations

* [Kubernetes](./kubernetes.md)

## Labels

Labels are filters that can be used to apply a role to one or more destinations.

Labels can be provided when registering new destinations through the `labels` configuration field.

See [Configuration](../configuration.md) for more information on configuring destinations.

### Semantics

Values within the same list will be combined to create a single filter, i.e. `AND` semantics.

```yaml
destinations:
  - labels:
    - kubernetes
    - us-west-1
```

In this case, only destinations that have both `kubernetes` and `us-west-1` labels will be matched.

Multiple destinations can be used to create multiple filters, i.e. `OR` semantics.

```yaml
destinations:
  - labels: [us-west-1]
  - labels: [us-east-1]
```

In this example, destinations having either `us-west-1` or `us-east-1` labels will be matched.

### Automatic Labels

* Kind, e.g. `kubernetes`
