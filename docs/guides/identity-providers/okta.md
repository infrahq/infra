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
2. Navigate to **Applications > Applications**

![Create Application](../../images/okta-setup/connect-users-okta-okta1.png)

3. Create an Okta App:
  - Click **Create App Integration**.
  - Select **OIDC – OpenID Connect** and **Web Application**.
  - Click **Next**.

![App Type](../../images/okta-setup/connect-users-okta-okta2.png)

4. Configure your new Okta App:
  - For **App integration name** write **Infra**.
  - Under **General Settings** > **Grant type** select **Authorization Code** and **Refresh Token**.
  - For **Sign-in redirect URIs** add `http://localhost:8301`

![General Tab](../../images/okta-setup/connect-users-okta-okta4.png)

<details>
  <summary><strong>(Optional) Configure Okta for Infra Dashboard Login</strong></summary>

Add an additional redirect URI: `<your infra host>/login/callback`.

Examples:
  - `https://infra.company.internal/login/callback` (If infra is hosted at `infra.company.internal`)
  - `http://localhost/login/callback` if trying out Infra locally

</details>
<br />

5. For **Assignments** select the groups which will have access through Infra.
  
    Click **Save**.

6. While still on the screen for the application you just created navigate to the **Sign On** tab.
   On the **OpenID Connect ID Token** select **Edit**.
   Update the **Groups claim filter** to `groups` `Matches regex` `.*`.
   Click **Save**.

7. Copy the **URL**, **Client ID** and **Client Secret** values and provide them into Infra's Dashboard or CLI.
![Sign On](../../images/okta-setup/connect-users-okta-okta5.png)
