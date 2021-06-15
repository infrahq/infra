# Okta

## Contents

* [Prerequisites](#prerequisites)
* [Setup](#setup)
    * [Configure Okta Login](#configure-okta-login)
    * [Configure Okta Directory Sync](#configure-okta-directory-sync)
    * [Configure Infra Engine](#configure-infra-engine)
* [Usage](#usage)
    * [Log in with Okta](#log-in-with-okta)
    * [List Okta users](#list-okta-users)

## Prerequisites

* [Install Infra](../README.md#install)
* An Okta administrator account to create an Okta Application and will be used to create a read-only Okta user for Infra to use


## Setup

### Configure Okta Login 

1. Log into Okta as an Administrator
2. Under the left menu click **Applications > Applications**. Click **Create App Integration** then select **OIDC – OpenID Connect** and **Web Application**, then click **Next**.

![image](https://user-images.githubusercontent.com/3325447/119013012-90ac2100-b964-11eb-9266-b5f3ab3b7392.png)


3. For **App integration name** write **Infra**. For **Sign-in redirect URIs** write `http://localhost:8301`. Click **Save**.

![image](https://user-images.githubusercontent.com/3325447/119013336-e1237e80-b964-11eb-983d-dbc60fff5ab5.png)

4. On the **General** tab, **note** the **Client ID** and **Client Secret** for the next step.

![image](https://user-images.githubusercontent.com/3325447/119013461-ff897a00-b964-11eb-9365-cdf5d06cd9cd.png)

### Configure Okta directory sync with a read-only token

1. Create a new user by navigating to **Directory > People**, clicking **Add Person**.

![image](https://user-images.githubusercontent.com/3325447/119013798-55f6b880-b965-11eb-9889-e59797662df6.png)

2. Name this person First Name: **Infra** Last Name: **Read-only**. For username and email choose a shared team email such as contact@company.com.
3. Choose **Set by admin** for the password, and create a secure password for this user.
4. Navigate to **Security > Administrators**. Click **Add Administrator**, select the read-only **Infra** user, and check the **Read Only Administrator** checkbox. Then click **Add Administrator**.

Then, create a API token for this read-only user:

1. Log in as this new user
2. Navigate to **Security > API**, then click the **Tokens** tab.
3. Create a new Token by clicking **Create Token**. Name it **Infra**

![image](https://user-images.githubusercontent.com/3325447/119014216-bc7bd680-b965-11eb-81db-24f53354291c.png)

4. Note this token for the next step.

### Configure Infra

```
infra providers create okta \
    --api-token 00_aj082hjd018j2dalskdnvbpp7bqf4bsadkfjbsdufh \
    --domain example.okta.com \
    --client-id 0oapn0qwiQPiMIyR35d6 \
    --client-secret vU-bIjeFyMB7j_jd178HahIsd1oaIaspnuU
```

### List Okta users

```
$ infra users ls
EMAIL              	  PROVIDERS	  CREATED               ROLE
jeff@example.com  	  okta    	  About a minute ago    infra.owner
michael@example.com*	okta    	  About a minute ago	  infra.member
elon@example.com   	  okta    	  About a minute ago	  infra.member
tom@example.com    	  okta    	  About a minute ago	  infra.member
mark@example.com   	  okta    	  About a minute ago    infra.member
```

### Log in with Okta

```
$ infra login infra.example.com
? Choose a login provider  [Use arrows to move, type to filter]
> Okta [example.okta.com]
✔ Logging in with Okta... success
✔ Logged in...
✔ Kubeconfig updated
```
