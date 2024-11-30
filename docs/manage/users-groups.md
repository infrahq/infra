# Users & groups

## Users

### Listing users

#### CLI

To see all users managed by Infra, use `infra users list`:

```bash
infra users list
```

You'll see the resulting list of users:

```bash
NAME                         LAST SEEN
fisher@infrahq.com           just now
jeff@infrahq.com             5 mintues ago
matt.williams@infrahq.com    3 days ago
michael@infrahq.com          3 days ago
```

#### Dashboard

To see all users managed by Infra, navigate to **Users**.
![View users](../images/viewusers.png)

### Adding a user

#### CLI

To add a user to Infra, use `infra users add`:

```bash
infra users add example@acme.com
```

They will receive an email to set their password and login to the system.

#### Dashboard

To add a user to Infra, navigate to **Users** and click the **Add User** button. Enter the users email address. They will receive an email to set their password and login to the system.

### Removing a user

#### CLI

```bash
infra users remove example@acme.com
```

#### Dashboard

Navigate to **Users**. To the right of each user is an ellipses button (three dots). Click it and click **Remove user**.
![Remove user](../images/removeuser.png)

### Resetting a user's password

```bash
infra users edit example@acme.com --password
```

## Groups

### Listing groups

#### CLI

To see all groups managed by Infra including both Infra managed groups and Identity Provider Provided groups, use `infra groups list`:

```bash
infra groups list
```

You'll see the resulting list of groups with each of its users:

```bash
NAME        LAST SEEN
developers  alice@infrahq.com, bob@infrahq.com
design      carol@infrahq.com, david@infrahq.com
everyone    alice@infrahq.com, bob@infrahq.com, carol@infrahq.com
            david@infrahq.com
```

#### Dashboard

Navigate to **Groups**. All the groups managed by Infra will be shown.
![List groups](../images/listgroups.png)

### Creating a new group

#### CLI

To add a new group, use `infra groups add`:

```bash
infra groups add developers
```

#### Dashboard

Navigate to **Groups** and click the **Add group** button.

### Removing a group

#### CLI

To remove a group, use `infra groups remove`:

```bash
infra groups remove developers
```

#### Dashboard

Navigate to **Groups** and choose a group. Click the **Remove Group** button at the top.

### Adding a user to a group

#### CLI

To add a user to a group, use `infra groups adduser`:

```bash
infra groups adduser example@acme.com
```

#### Dashboard

Navigate to **Groups** and click on a group. Enter a user email in the text box and click the **Add User** button.

### Removing a user from a group

#### CLI

To remove a user from a group, use `infra groups removeuser`:

```bash
infra groups removeuser example@acme.com developers
```

#### Dashboard

Navigate to **Groups** and click on a group. Click **Remove** to the right of any user.
![Remove group](../images/removeuserfromgroup.png)
