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
* An Okta administrator account


## Setup

### Configure Okta Login

1. Log into Okta as an Administrator
2. Under the left menu click **Applications > Applications**. Click **Create App Integration** then select **OIDC – OpenID Connect** and **Web Application**, then click **Next**.
3. For **App integration name** write **Infra**. Optionally: add [the infra logo](./docs/images/okta.png). For **Login redirect URIs** write `http://localhost:8301`. Click **Save**.
4. On the **General** tab, note the **Client ID** and **Client Secret** for the next step.

Store the client secret as a Kubernetes secret for Infra to read:

### Configure Okta directory sync with a read-only token

1. Create a new user by navigating to **Directory > People**, clicking **Add Person**.
2. Name this person First Name: **Infra** Last Name: **Read-only**. For username and email choose a shared team email such as contact@company.com.
3. Choose **Set by admin** for the password, and create a secure password for this user.
4. Navigate to **Security > Administrators**. Click **Add Administrator**, select the read-only **Infra** user, and check the **Read Only Administrator** checkbox. Then click **Add Administrator**.

Then, create a API token for this read-only user:

1. Log in as this new user
2. Navigate to **Security > API**, then click the **Tokens** tab.
3. Create a new Token by clicking **Create Token**. Name it **Infra**
4. Note this token for the next step.

### Configure Infra Engine

Add secrets from the previous step:

```
$ kubectl -n infra create secret generic infra \
    --from-literal="okta-client-secret=In6P_qEoEVugEgk_7Z-Vkl6CysG1QapBBCzS5O7m" \
    --from-literal="okta-api-token=00nQtyRYAXOaA03xRJ5Ok2o6Tg8f19ku9DD3ySS8U9"
```

Then update Infra Engine's configuration:

```yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: infra
  namespace: infra
data:
  infra.yaml: |
    providers:
      okta:
        domain: acme.okta.com                                 # REPLACE ME: Your Okta domain
        client-id: 0oapn0qwiQPiMIyR35d6                       # REPLACE ME: Your Client ID
        client-secret: /var/run/infra/secrets/okta-client-secret
        api-token: /var/run/infra/secrets/okta-api-token

    permissions:
      - user: michael@acme.com
        permission: admin
EOF
```

Finally, rollout a new version of Infra to reflect the new configuration:

```
$ kubectl rollout restart -n infra statefulset/infra
```

## Usage

### Log in with Okta

```
$ infra login infra.acme.com
✔ Logging in with Okta... success
✔ Logged in as michael@acme.com
✔ Kubeconfig updated
```

### List Okta users

```
$ infra users ls
USER            	EMAIL              	CREATED         PROVIDERS  	PERMISSION	      	
usr_cHHfCsZu3by7	michael@infrahq.com	3 minutes ago   okta     	view      	
usr_jojpIOMrBM6F	elon@infrahq.com   	3 minutes ago   okta     	view      	
usr_mBOjQx8RjC00	mark@infrahq.com   	3 minutes ago   okta     	view      	
usr_o7WreRsehzyn	tom@infrahq.com    	3 minutes ago   okta     	view      	
usr_uOQSaCwEDzYk	jeff@infrahq.com   	3 minutes ago  	okta     	view  
```