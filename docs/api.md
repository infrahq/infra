# API Reference

## Generating a new API key
In order to generate a new API key you must first have an existing API key with the `infra.api-keys.create` permission. By default a root key with this permission is created in the Infra Registry. To retrieve the root Infra API key get the `infra-registry.rootApiKey` secret from your Infra Registry Kubernetes deployment.

```
export INFRA_ROOT_API_KEY=$(kubectl --namespace infrahq get secrets infra-registry -o jsonpath='{.data.rootApiKey}' | base64 -D)
```

Use this API key to create a new API key with some specified permissions by sending a request to the Infra Registry API.
```
curl --request POST \
  --url https://$INFRA_REGISTRY_ADDRESS/v1/api-keys \
  --header 'Authorization: Bearer $INFRA_ROOT_API_KEY' \
  --header 'Content-Type: application/json' \
  --data '{
  "name": "example-api-key",
  "permissions": [
    "users.read"
  ]
}'
```
