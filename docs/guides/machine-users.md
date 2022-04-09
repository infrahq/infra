# Machine users

For accessing infrastructure via CI/CD jobs (e.g. GitHub Actions or similar), Infra can grant access to **machine users**.

## Creating a machine user

To create a machine user, run `infra id add` with a non-email name (e.g. `bot`):

```
infra id add bot
```

## Creating an access key for this machine user

Next, we'll create a key for users 

```
infra key add first-key bot
```

You'll receive a key in the output:

```bash
key: WV0x7IIAhc.7zAYrf3f4QnZAnCueZI3RX8v
```

## Logging in as a machine user

To log in as a machine user, you can pass their access key via the `--key` flag to `infra login`:

```
infra login infra.acme.com --key WV0x7IIAhc.7zAYrf3f4QnZAnCueZI3RX8v
```

After logging in, infrastructure (e.g. a kubernetes cluster named `staging`) can be accessed as `bot`:

```
infra use kubernetes.staging

# example: deploy an app
kubectl apply -f application.yaml
```

## Rotating machine user keys

To view a list of keys that have been created:

```
infra keys list
```

To revoke a key:

```
infra keys remove first-key
```

Lastly, create a new key for `bot`:

```
infra keys add new-key bot
```
