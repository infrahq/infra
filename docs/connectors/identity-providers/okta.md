# Okta

## Connecting Okta

To connect Okta, run the following command:

```bash
infra providers add okta \
  --url <your okta url (or domain)> \
  --client-id <your okta client id> \
  --client-secret <your okta client secret>
```


## Finding required values

1. Login to the Okta dashboard.
2. Under the left menu click **Applications > Applications**.  
   Click **Create App Integration**.  
   Select **OIDC – OpenID Connect** and **Web Application**.  
   Click **Next**.

![Create Application](../../images/connect-users-okta-okta1.png)

3. For **App integration name** write **Infra**.  
   In **General Settings** > **Grant type** select **Authorization Code** and **Refresh Token**.  
   For **Sign-in redirect URIs** write `http://localhost:8301`. For **Assignments** select the groups which will have access through Infra.  
   Click **Save**.

![App Type](../../images/connect-users-okta-okta2.png)

4. On the **General** tab, **note** the **Client ID**, **Client Secret**, and **Okta domain** for adding your Okta information to Infra later.

![General Tab](../../images/connect-users-okta-okta4.png)

1. While still on the screen for the application you just created navigate to the **Sign On** tab.  
   On the **OpenID Connect ID Token** select **Edit**.  
   Update the **Groups claim filter** to `groups` `Matches regex` `.*`.  
   Click **Save**.

![Sign On](../../images/connect-users-okta-okta5.png)
