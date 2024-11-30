# Using Infra

Get started by [downloading](./download.md) the Infra CLI.

## Logging in

Log in to Infra. `infra login` will prompt for which server to log in to.

```
infra login
```

## Listing your access

Once logged in, Infra provides a command, `infra list` to list the infrastructure the current user can access:

```
infra list
```

Example output:

```
  NAME                   ACCESS
  development            edit
  development.frontend   admin
  production             view,logs
```

## Accessing infrastructure

Infra automatically keeps local configuration files (e.g. KubeConfig, SSH config) up to date. The Infra CLI includes a command, `infra use`, to switch the local context to a specific resource. For example to switch to the cluster named `development`:

```
infra use development
```

However, Infra is also compatible with existing tooling (e.g. kubectl). Contexts are prefixed with `infra:`

```
kubectl --context infra:development get pods
```

## Viewing the current user

To see the currently logged-in user, run `infra info`:

```
infra info
```

Important login information for the current user will be shown:

```
            Server: acme.infrahq.com
              User: jeff@acme.co (dz9jbzSsJa)
 Identity Provider: Google (accounts.google.com)
            Groups: Engineering
```
