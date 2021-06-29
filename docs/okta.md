# Okta

## Contents

* [Prerequisites](#prerequisites)
* [Setup](#setup)
    * [Create an Okta App](#create-an-okta-app)
    * [Add Okta information to Infra registry](#add-okta-information-to-infra-registry)
* [Usage](#usage)
    * [Log in with Okta](#log-in-with-okta)
    * [List Okta users](#list-okta-users)

## Prerequisites

* [Install Infra](../README.md#install)

## Setup

### Create an Okta App 

1. Log into Okta as an Administrator
2. Under the left menu click **Applications > Applications**. Click **Create App Integration** then select **OIDC – OpenID Connect** and **Web Application**, then click **Next**.

![image](https://user-images.githubusercontent.com/3325447/119013012-90ac2100-b964-11eb-9266-b5f3ab3b7392.png)

3. For **App integration name** write **Infra**. For **Sign-in redirect URIs** write `http://localhost:8301`. Click **Save**.

![image](https://user-images.githubusercontent.com/3325447/122437369-a57dd380-cf67-11eb-871b-3f1d2482c6c2.png)

4. On the **General** tab, **note** the **Client ID** and **Client Secret** for the next step.

![image](https://user-images.githubusercontent.com/3325447/122437934-2dfc7400-cf68-11eb-805f-745d0677bb89.png)

5. Navigate to **Security > API**, then click the **Tokens** tab. Create a new Token by clicking **Create Token**. Name it **Infra**

![image](https://user-images.githubusercontent.com/3325447/119014216-bc7bd680-b965-11eb-81db-24f53354291c.png)

### Add Okta information to Infra registry

```
infra source create okta \
    --api-token 00_aj082hjd018j2dalskdnvbpp7bqf4bsadkfjbsdufh \
    --domain example.okta.com \
    --client-id 0oapn0qwiQPiMIyR35d6 \
    --client-secret vU-bIjeFyMB7j_jd178HahIsd1oaIaspnuU
```

### List Okta users

```
$ infra user list
EMAIL              	  SOURCES	  CREATED               ADMIN
jeff@example.com  	  okta    	  About a minute ago
michael@example.com*  okta    	  About a minute ago
elon@example.com   	  okta    	  About a minute ago
tom@example.com    	  okta    	  About a minute ago
mark@example.com   	  okta    	  About a minute ago
admin@example.com     infra       5 minutes ago         x
```

### Log in with Okta

```
$ infra login <INFRA_REGISTRY_EXTERNAL_IP>
? Choose a login method  [Use arrows to move, type to filter]
> Okta [example.okta.com]
✔ Logging in with Okta... success
✔ Logged in...
✔ Kubeconfig updated
```
