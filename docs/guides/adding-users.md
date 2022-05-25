# Adding & Removing Users

## Adding users

To add a user to Infra, use `infra id add`:

```
infra users add example@acme.com
```

You'll be provided a one time password to share with the user (via slack, eamil or similar) they should use when running `infra login`.

## Removing users

```
infra users remove example@acme.com
```

## Listing users

To see all users being managed by Infra, use `infra id list`:

```
infra users list
```

You'll see the resulting list of users:

```
NAME                         LAST SEEN
fisher@infrahq.com           just now
jeff@infrahq.com             5 mintues ago
matt.williams@infrahq.com    3 days ago
michael@infrahq.com          3 days ago
```
