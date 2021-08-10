# Okta

## Contents

* [Prerequisites](#prerequisites)
* [Setup](#setup)
    * [Create an Okta App](#create-an-okta-app)
    * [Add Okta information to Infra registry](#add-okta-information-to-infra-registry)
* [Usage](#usage)
    * [Login with Okta](#log-in-with-okta)
    * [List Okta users](#list-okta-users)

## Prerequisites

* [Install Infra](../README.md#install)

## Setup

### Create an Okta App 

1. Login to the Okta administrative dashboard.
2. Under the left menu click **Applications > Applications**. Click **Create App Integration** then select **OIDC – OpenID Connect** and **Web Application**, and click **Next**.

![okta_applications](https://user-images.githubusercontent.com/5853428/124651126-67c9e780-de4f-11eb-98bd-def34bea95fd.png)
![okta_create_new_app](https://user-images.githubusercontent.com/5853428/124651919-60efa480-de50-11eb-9cb8-226f1c225191.png)

3. For **App integration name** write **Infra**. For **Sign-in redirect URIs** write `http://localhost:8301`. Click **Save**.

![okta_new_web_app_integration](https://user-images.githubusercontent.com/5853428/124652225-b88e1000-de50-11eb-8da3-36af6ba28bd8.png)

4. On the **General** tab, **note** the **Client ID** and **Client Secret** for the next step. Note the **Okta domain** for adding your Okta information to Infra registry later.

![okta_application](https://user-images.githubusercontent.com/5853428/125355241-a3febb80-e319-11eb-8fc6-84df2509f621.png)

5. Navigate to **Security > API**, then click the **Tokens** tab. Create a new Token by clicking **Create Token**. Name it **infra**.

![okta_create_token](https://user-images.githubusercontent.com/5853428/124652451-0276f600-de51-11eb-9d22-92262de76371.png)
![okta_api_token](https://user-images.githubusercontent.com/5853428/124652864-787b5d00-de51-11eb-81d8-e503babfdbca.png)

### Add Okta information to Infra registry

Edit your [Infra configuration](./configuration.md) (e.g. `infra.yaml`) to include an Okta source:

```yaml
sources:
  - type: okta
    domain: acme.okta.com
    clientId: 0oapn0qwiQPiMIyR35d6
    clientSecret: jfpn0qwiQPiMIfs408fjs048fjpn0qwiQPiMajsdf08j10j2
    apiToken: 001XJv9xhv899sdfns938haos3h8oahsdaohd2o8hdao82hd

users:
  - name: admin@example.com
    roles:
      - name: admin
        kind: cluster-role
        clusters:
          - cluster-AAA
          - cluster-BBB
  - name: developer@example.com
    roles:
      - name: writer
        kind: cluster-role
        clusters:
          - cluster-AAA
```

Then apply this config change:

```
helm upgrade infra infrahq/infra --set-file config=./infra.yaml --recreate-pods
```

### List users

```
$ infra users
EMAIL                 CREATED               ADMIN
jeff@example.com      About a minute ago
michael@example.com   About a minute ago
elon@example.com.     About a minute ago
tom@example.com       About a minute ago
mark@example.com      About a minute ago
admin@example.com     5 minutes ago         x
```

### Login with Okta

```
$ infra login <INFRA_REGISTRY_EXTERNAL_IP>
? Choose a login method  [Use arrows to move, type to filter]
> Okta [example.okta.com]
✔ Logging in with Okta... success
✔ Logged in...
✔ Kubeconfig updated
```
