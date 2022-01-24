# Okta

## Configure

| Parameter       | Description                 |
|-----------------|-----------------------------|
| `domain`        | Okta domain                 |
| `clientID`      | Okta client ID              |
| `clientSecret`  | Okta client secret          |

## Connect an Okta Provider

This guide will walk you through the process of setting up Okta as an identity provider for Infra. At the end of this process you will have updated your Infra configuration with an Okta provider that looks something like this:

```
providers:
  - kind: okta
    domain: acme.okta.com
    clientID: 0oapn0qwiQPiMIyR35d6
    clientSecret: kubernetes:infra-okta/clientSecret
```

## Create an Okta App

1. Login to the Okta administrative dashboard.
2. Under the left menu click **Applications > Applications**.  
Click **Create App Integration**.  
Select **OIDC – OpenID Connect** and **Web Application**.  
Click **Next**.

![okta_applications](https://user-images.githubusercontent.com/5853428/124651126-67c9e780-de4f-11eb-98bd-def34bea95fd.png)
![okta_create_new_app](https://user-images.githubusercontent.com/5853428/124651919-60efa480-de50-11eb-9cb8-226f1c225191.png)

3. For **App integration name** write **Infra**.  
In **General Settings** > **Grant type** select **Authorization Code** and **Refresh Token**.  
For **Sign-in redirect URIs** write `http://localhost:8301`. For **Assignments** select the groups which will have access through Infra.  
Click **Save**.

![okta_app_creation_group_assignment](https://user-images.githubusercontent.com/5853428/130118354-b7ebeee7-4b7b-41cf-a022-ad165fa6f5db.png)


4. On the **General** tab, **note** the **Client ID**, **Client Secret**, and **Okta domain** for adding your Okta information to Infra later.

![okta_application](https://user-images.githubusercontent.com/5853428/125355241-a3febb80-e319-11eb-8fc6-84df2509f621.png)

5. While still on the screen for the application you just created navigate to the **Sign On** tab.  
On the **OpenID Connect ID Token** select **Edit**.  
Update the **Groups claim filter** to `groups` `Matches regex` `.*`.  
Click **Save**.

![groups_claim](https://user-images.githubusercontent.com/5853428/150852764-9a447ab5-7e24-483d-86e3-cd2767b07b56.png)

### Add the Okta client secret to the Infra deployment

The Okta client secret is sensitive information which should not be stored in the Infra configuration file. In order for Infra to access this secret values it should be stored in a secret provider, for this example we will use Kubernetes Secret objects **in the same namespace that the Infra is deployed in**.

Create [a Kubernetes Secret object](https://kubernetes.io/docs/tasks/configmap-secret/) to store the Okta client secret (noted in step 4 of `Create an Okta App`). You can name this Secret as you desire, this name will be specified in the Infra configuration.

#### Example Secret Creation

There are [many ways to store secrets](../secrets.md). Here's an example of using Kubernetes for the secret storage.

Store the Okta client secret using a Kubernetes Secret object in the namespace that Infra is running in.
```
$ OKTA_CLIENT_SECRET=jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2

$ kubectl -n infrahq create secret generic infra-okta --from-literal=clientSecret=$OKTA_CLIENT_SECRET
```

see [secrets.md](../secrets.md) for further details.

## Add Okta Information to Infra Configuration

Edit your [Infra configuration](./configuration.md) (e.g. `infra.yaml`) to include an Okta provider:

```yaml
# infra.yaml
---
providers:
  - kind: okta
    domain: example.okta.com
    clientID: 0oapn0qwiQPiMIyR35d6
    clientSecret: kubernetes:infra-okta/clientSecret  # <secret kind>:<secret name>
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
  providers:
    - kind: okta
      domain: example.okta.com
      clientID: 0oapn0qwiQPiMIyR35d6
      clientSecret: kubernetes:infra-okta/clientSecret  # <secret kind>:<secret name>
```

Then apply this config change:

```
helm -n infrahq upgrade -f values.yaml infra infrahq/infra
```

