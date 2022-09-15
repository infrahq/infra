---
title: Working with Groups
position: 4
---

# Working with Groups

## Listing groups

To see all groups being managed by Infra including both Infra managed groups and Identity Provider Provided groups, use `infra groups list`:

```
infra groups list
```

You'll see the resulting list of groups with each of its users:

```
NAME        LAST SEEN
developers  alice@infrahq.com, bob@infrahq.com
design      carol@infrahq.com, david@infrahq.com
everyone    alice@infrahq.com, bob@infrahq.com, carol@infrahq.com
            david@infrahq.com
```

## Creating a new group

To add a new group, use `infra groups add`:

```
infra groups add developers
```

## Removing a group

To remove a group, use `infra groups remove`:

```
infra groups remove developers
```

## Adding a user

To add a user to a group, use `infra groups adduser`:

```
infra groups adduser example@acme.com
```

## Removing a user

To remove a user from a group, use `infra groups removeuser`:

```
infra groups removeuser example@acme.com developers
```

