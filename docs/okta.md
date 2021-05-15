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

* An Okta administrator account

## Setup

### Configure Okta Login

1. Log into Okta as an Administrator
2. Under the left menu click **Applications > Applications**. Click **Add Application** then **Create New App**. Select "OpenID Connect" from the dropdown, then click **Create**
3. For **Application name** write **Infra**. Optionally: add [the infra logo](./docs/images/okta.png). For **Login redirect URIs** write `http://localhost:8301`. Click **Save**.
4. Under the **Assignments** tab, assign Infra to one or more users or groups.
5. Back to the **General** tab, note the **Client ID** and **Client Secret** for the next step.

Store the client secret as a Kubernetes secret for Infra to read:

### Configure Okta Directory Sync

Optionally, create a read-only user:

1. Create a new user by navigating to **Directory > People**, clicking **Add Person**.
2. Name this person First Name: **Infra** Last Name: **Read-only**. For username and email choose a shared team email such as contact@acme.com.
3. Choose **Set by admin** for the password, and create a secure password for this user.
4. Navigate to **Security > Administrators**. Click **Add Administrator**, select the read-only **Infra** user, and check the **Read Only Administrator** checkbox. Then click **Add Administrator**.
5. Log in as this new user

Then, create an API key for Infra to synchronize users:

1. Navigate to **Security > API**, then click the **Tokens** tab.
2. Create a new Token by clicking **Create Token**. Add this token to Infra as a secret t oread

### Configure Infra Engine

Add secrets from the previous step:

```
$ kubectl -n infra edit secret infra \
    --from-literal=okta-client-secret=In6P_qEoEVugEgk_7Z-Vkl6CysG1QapBBCzS5O7m \   # Client Secret
    --from-literal=okta-api-token=00nQtyRYAXOaA03xRJ5Ok2o6Tg8f19ku9DD3ySS8U9       # API Token
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
        client-secret: /etc/secrets/infra/okta-client-secret
        api-token: /etc/secrets/infra/okta-api-token
    permissions:                                              # All users get view permissions by default
      - user: michael@acme.com                                # EXAMPLE: Give a single user admin permission
        permission: admin
EOF
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
USER ID         	PROVIDERS	EMAIL             CREATED     	  PERMISSION
usr_vfZjSZctMptn	okta     	michael@acme.com  2 minutes ago   admin
usr_mvm8YVTvOGY4	okta     	tom@acme.com      2 minutes ago	  view      	
usr_Lgr5pQIkrrM4	okta     	elon@acme.com     2 minutes ago	  view      	
usr_g7UcCe7CUdHA	okta     	jeff@acme.com     2 minutes ago	  view      	   	
usr_wH4Oc9QdPxpj	okta     	mark@acme.com     2 minutes ago	  view  
```