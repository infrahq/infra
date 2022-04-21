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

To see all users being managed by Infra, use `infra id list`:

```
infra id list
```

You'll see the resulting list of users:

```
NAME                       TYPE
fisher@infrahq.com         user
jeff@infrahq.com           user
matt.williams@infrahq.com  user
michael@infrahq.com        user
bot                        machine
connector                  machine
```
