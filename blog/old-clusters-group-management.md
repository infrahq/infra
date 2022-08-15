---
title: Identify old clusters and group management
author: Matt Williams
date: 2022-07-08
---

![Last Seen](./images/changelog-0.13.6-lastseenconnected-ui.png)

**Infra 0.13.6 was just released**. New features include managing groups in the CLI, the ability to easily identify older, stale clusters in both the UI and the CLI, and feedback around when the connector keys will expire.

### Identity older, stale clusters

As you work with Infra, you may find that you added clusters that no longer exist. Figuring out which clusters haven't been connected to in a while may be awkward to determine, so we now track and display when the connector was last seen by the server, as well as whether it is currently connected. This feature was added to both the UI and the CLI.

![Last Seen](./images/changelog-0.13.6-lastseenconnected.png)

![Last Seen UI](./images/changelog-0.13.6-lastseenconnected-ui.png)

### Groups

When using the CLI, you will notice there is a new **Groups** command. This will allow you to create and delete local groups, and add and remove users from those groups. This feature does not affect any groups defined in your IdP and are only for local groups.

![Infra Groups](./images/changelog-0.13.6-infragroups.png)

### `Infra-Version` header

Before going into those features, its important to note that there is a breaking change if you are directly using the Infra API. The **Infra-Version header** is now required for all API requests which will help with future migrations. If you are not using the API in your own applications, you don't need to worry about this.

There are also a number of other new features in the UI, CLI, as well as the API. Review the Changelog for all the details [here](https://github.com/infrahq/infra/releases/tag/v0.13.6)
