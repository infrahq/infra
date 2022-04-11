
# Managing Infra Admins

## Built-in Infra Roles

Infra has built-in roles for promoting users to manage Infra.

* **admin**: Full admin access to Infra
* **user**: List and access infrastructure

## Promoting a user to an Infra admin

This will provide `admin@example.com` full access to the Infra API, including creating additional grants, managing identity providers, managing destinations, and managing other users.

```
infra grants add admin@example.com infra --role admin
```

## Setting a group to an Infra admin

```
infra grants add --group engineering infra --role admin
```

## Revoking admin access

```
infra grants remove admin@example.com infra --role admin
```

## Giving a user limited access to Infra

This Grant will provide `dev@example.com` *some* access to the Infra API, including logging in and using a destination they have been granted access to, listing destinations, and updating their own user. It does *not* include access to creating additional grants, managing identity providers, managing destinations, or managing other users.

```
infra grants add dev@example.com infra --role user
```
