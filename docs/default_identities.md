# Default Identities

By default the Infra server will have two machine identities that use access keys to authenticate.

## Admin Identity
The admin is the default root identity for interacting with Infra server. It is granted Infra admin role and can call all Infra endpoints. It should be used during initial setup, then its credentials should be stored securely in case it is needed in the future.

To authenticate as the admin identity use the default admin access key. If this key is not provided by the user during Helm install, it can be retrieved through following the setup process.

WARNING: This admin access key grants full access to Infra. Do not share it. Consider backing this value up in a secure place, such as a secret manager, to use for emergency access only.

This key can then be used to login to Infra.
```
$ infra login $INFRA_SERVER       
  Logging in to $INFRA_SERVER
? Select a login method: Login with Access Key
? Your Access Key: ***********************************
  Logged in as admin
```

## Connector Identity
This is the identity associated with the default Infra connector created on installation of the Infra server. It is the identity and access key the default connector uses to connect the the Infra server. The default connector access key can be specified in the Helm install, otherwise it is randomly generated.
