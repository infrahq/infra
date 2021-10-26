# API Reference

## Generating a new API key

In order to generate a new API key you must first have an existing API key with the `infra.api-keys.create` permission. By default a root key with this permission is created in Infra. To retrieve the root Infra API key get the `infra/root-key` secret from your Infra Kubernetes deployment.

```bash
INFRA_ROOT_API_KEY=$(kubectl --namespace infrahq get secrets infra -o jsonpath='{.data.root-key}' | base64 --decode)
```

Use this API key to create a new API key with some specified permissions by sending a request to the Infra API.

```bash
curl --request POST \
  --header 'Authorization: Bearer $INFRA_ROOT_API_KEY' \
  --header 'Content-Type: application/json' \
  --data '{"name": "example-api-key", "permissions": ["users.read"]}'
  https://$INFRA_HOST/v1/api-keys
```
