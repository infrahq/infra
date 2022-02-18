# API Reference

## Calling the Infra API
If you have an access key you can call the Infra API directly by setting the authorization header in a request to your issued access key.
```bash
curl --request GET \
  --url https://$INFRA_SERVER/v1/$ENDPOINT \
  --header "Authorization: Bearer $YOUR_ACCESS_KEY"
```

If the identity associated with this access key has been granted the required permission for this endpoint the call will succeed.

## Create an Access Key
In order to create an access key you may need to:
- Have access to an identity with the permissions needed.
    - `infra.machine.create` this access key should be able to create new machines.
    - `infra.grant.create` if the access key should be able to grant additional permissions.
    - `infra.accesskey.create` if the access key should be able to create other access keys.
- Create or use an existing identity.
- Create an access key that an identity uses for authentication (for API calls). 

### CLI Example
This concrete example shows the steps of creating an API client that manages identity providers.

1. Login to the Infra server with an identity that has the permissions to create machine identities and access keys. This example uses [the default admin identity](./default_identities.md).
```
$ infra login $INFRA_SERVER         
  Logging in to $INFRA_SERVER
? Select a login method: Login with Access Key
? Your Access Key: ***********************************
  Logged in as admin
```

2. Create the machine identity that will be calling the Infra API.
```
$ infra machines create idp_automation --permissions="infra.provider.*"
```

3. Create an access key for this machine identity to use as authentication when calling the API.
```
$ infra keys create idp_automation_authn idp_automation
key: sUpEr.SeCrEtVaLuE
```

This access key can be used to call the API as the "IDP automation" client.
```bash
curl --request GET \
  --url https://$INFRA_SERVER/v1/providers \
  --header 'Authorization: Bearer sUpEr.SeCrEtVaLuE'
```
