# Sources / Okta

## Configure Okta Source

| Parameter      | Description                                  |
|----------------|----------------------------------------------|
| `domain`       | Okta domain                                  |
| `clientId`     | Okta client ID                               |
| `clientSecret` | Okta client secret                           |
| `apiToken`     | Okta API token                               |

## Connect an Okta Source

This guide will walk you through the process of setting up Okta as an identity provider for Infra. At the end of this process you will have updated your Infra configuration with an Okta source that looks something like this:

```
sources:
  - kind: okta
    domain: acme.okta.com
    clientId: 0oapn0qwiQPiMIyR35d6
    clientSecret: infra-registry-okta/clientSecret
    apiToken: infra-registry-okta/apiToken
```

## Create an Okta App

1. Login to the Okta administrative dashboard.
2. Under the left menu click **Applications > Applications**. Click **Create App Integration** then select **OIDC – OpenID Connect** and **Web Application**, and click **Next**.

![okta_applications](https://user-images.githubusercontent.com/5853428/124651126-67c9e780-de4f-11eb-98bd-def34bea95fd.png)
![okta_create_new_app](https://user-images.githubusercontent.com/5853428/124651919-60efa480-de50-11eb-9cb8-226f1c225191.png)

3. For **App integration name** write **Infra**. For **Sign-in redirect URIs** write `http://localhost:8301`. For **Assignments** select the groups which will have access through Infra. Click **Save**.

![okta_app_creation_group_assignment](https://user-images.githubusercontent.com/5853428/130118354-b7ebeee7-4b7b-41cf-a022-ad165fa6f5db.png)


4. On the **General** tab, **note** the **Client ID**, **Client Secret**, and **Okta domain** for adding your Okta information to Infra registry later.

![okta_application](https://user-images.githubusercontent.com/5853428/125355241-a3febb80-e319-11eb-8fc6-84df2509f621.png)

5. Navigate to **Security > API**, then click the **Tokens** tab. Create a new Token by clicking **Create Token**. Name it **infra**. Note this token value for later.

![okta_create_token](https://user-images.githubusercontent.com/5853428/124652451-0276f600-de51-11eb-9d22-92262de76371.png)
![okta_api_token](https://user-images.githubusercontent.com/5853428/124652864-787b5d00-de51-11eb-81d8-e503babfdbca.png)

### Add Okta secrets to the Infra registry deployment
The Okta client secret and API token are sensitive information which cannot be stored in the Infra configuration file. In order for Infra to access these secret values they must be stored in Kubernetes Secret objects **in the same namespace that the Infra registry is deployed in**.

Create [Kubernetes Secret objects](https://kubernetes.io/docs/tasks/configmap-secret/) to store the Okta client secret and API token (noted in steps 4 and 5 of `Create an Okta App` respectively). You can name these Secrets as you desire, these names will be specified in the Infra configuration.

#### Example Secret Creation
Store the Okta client secret and API token on the same Kubernetes Secret object in the namespace that Infra registry is running in.
```
OKTA_CLIENT_SECRET=jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
OKTA_API_TOKEN=001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd
kubectl -n infrahq create secret generic infra-registry-okta --from-literal=clientSecret=$OKTA_CLIENT_SECRET --from-literal=apiToken=$OKTA_API_TOKEN
```

## Add Okta Information to Infra Configuration

Edit your [Infra configuration](./configuration.md) (e.g. `infra.yaml`) to include an Okta source:

```yaml
# infra.yaml
---
sources:
  - kind: okta
    domain: example.okta.com
    clientId: 0oapn0qwiQPiMIyR35d6
    clientSecret: infra-registry-okta/clientSecret  # <Kubernetes secret object>/<secret name>
    apiToken: infra-registry-okta/apiToken
```

Then apply this config change:

```
helm -n infrahq upgrade --set-file config=infra.yaml infra infrahq/infra
```

Infra configuration can also be added to Helm values:

```yaml
# values.yaml
---
config:
  sources:
    - kind: okta
      domain: example.okta.com
      clientId: 0oapn0qwiQPiMIyR35d6
      clientSecret: infra-registry-okta/clientSecret  # <Kubernetes secret object>/<secret name>
      apiToken: infra-registry-okta/apiToken
```

Then apply this config change:

```
helm -n infrahq upgrade -f values.yaml infra infrahq/infra
```

### Login with Okta

```
$ infra login <INFRA_REGISTRY_EXTERNAL_IP>
? Choose a login method  [Use arrows to move, type to filter]
> Okta [example.okta.com]
```
