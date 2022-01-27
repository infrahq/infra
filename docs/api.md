# API Reference

## Generating a new API token

In order to generate a new API token you must first have an existing API token with the `infra.apiToken.create` permission. By default a root key with this permission is created in Infra. To retrieve the root Infra API token get the `infra/root-api-token` secret from your Infra Kubernetes deployment.

```bash
INFRA_ROOT_API_TOKEN=$(kubectl --namespace infrahq get secrets infra -o jsonpath='{.data.root-api-token}' | base64 --decode)
```

Use this API token to create a new API token with some specified permissions by sending a request to the Infra API.

```bash
curl --request POST \
  --header 'Authorization: Bearer $INFRA_ROOT_API_TOKEN' \
  --header 'Content-Type: application/json' \
  --data '{"name": "example-api-token", "permissions": ["infra.user.read"]}'
  https://$INFRA_SERVER/v1/api-tokens
```
