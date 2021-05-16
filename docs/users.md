# Managing Users

## Contents

* [Listing Users](#listing-users)
* [Creating users](#creating-users)
* [Deleting users](#deleting-users)

## Listing Users

```
$ infra users ls
USER ID         	PROVIDERS	EMAIL             CREATED     	  PERMISSION
usr_vfZjSZctMptn	        	bob@acme.com      2 minutes ago   view
```

## Creating users

```
$ infra users create jeff@acme.com
usr_gja0ew4f8a0s
```

Verify the user has been created

```
$ infra users ls
USER ID         	PROVIDERS	EMAIL             CREATED     	  PERMISSION
usr_gja0ew4f8a0s	         	jeff@acme.com     3 seconds ago   view
usr_vfZjSZctMptn	         	bob@acme.com      2 minutes ago   view
```

## Deleting users

```
$ infra users delete usr_gja0ew4f8a0s
usr_gja0ew4f8a0s
```
