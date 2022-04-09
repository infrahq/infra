# Adding & Removing Users

## Adding users

To add a user to Infra, use `infra id add`:

```
infra id add example@acme.com
```

You'll be provided a one time password to share with the user (via slack, eamil or similar) they should use when running `infra login`.

## Removing users

```
infra id remove example@acme.com
```

## Listing users

To see all users being managed by Infra, use `infra list`:

```
infra list
```

You'll see the resulting list of users:

```
NAME (9)                   TYPE       PROVIDER
fisher@infrahq.com         user       okta
jeff@infrahq.com           user       okta
matt.williams@infrahq.com  user       okta
michael@infrahq.com        user       infra
bot                        machine    infra
connector                  machine    infra
```
