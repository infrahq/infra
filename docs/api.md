# API Reference

## Calling the Infra API
If you have an access-key you can call the Infra API directly by setting the authorization header in a request to your issued access-key.
```bash
curl --request GET \
  --url https://$INFRA_SERVER/v1/$ENDPOINT \
  --header "Authorization: Bearer $YOUR_ACCESS_KEY"
```

If the identity associated with this access-key has been granted the Infra admin role the call will succeed.

## Create an Access Key
In order to create an access key you may need to:
1. Have access to an identity with the Infra admin role.
1. Create an access key that an identity uses for authentication (for API calls). 

### CLI Example
This concrete example shows the steps of creating a new API client.

1. Login to the Infra server with an identity that has the Infra admin role. This example uses [the default admin identity](./default_identities.md).
```
$ infra login $INFRA_SERVER         
  Logging in to $INFRA_SERVER
? Select a login method: Login with Access Key
? Your Access Key: ***********************************
  Logged in as admin
```

2. Create the machine identity that will be calling the Infra API.
```
$ infra machines create example_machine
```

3. Grant the machine the Infra admin role.
```
$ infra access grant -m example_machine -r admin infra
```

3. Create an access key for this machine identity to use as authentication when calling the API.
```
$ infra keys create example_authn example_machine
key: sUpEr.SeCrEtVaLuE
```

This access key can be used to call the API as the "example_machine" client.
```bash
curl --request GET \
  --url https://$INFRA_SERVER/v1/providers \
  --header 'Authorization: Bearer sUpEr.SeCrEtVaLuE'
```
