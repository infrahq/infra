# Google

## Connecting Google
To connect Google, run the following command:

```bash
infra providers add google \
  --url accounts.google.com \
  --client-id <your google client id> \
  --client-secret <your google client secret>
```

## Finding required values

1. Login to the Google Cloud console.
2. Select the project you wish to create a client for or create a new project.

![Google Cloud project console](../../images/google-setup/connect-users-google-1.png)

3. If you have not yet configured OAuth consent for this project navigate to the **APIs and Services > OAuth consent screen** dashboard.

  ![OAuth consent navigation](../../images/google-setup/connect-users-google-2.png)

  - For **User Type** select **Internal** to only allow users from your own organization to use the application.
  - Click **Create**.
  - For **App information** enter an **App name** and a **User support email**.
  - For **Developer contact information** enter an email.
  - Click **Save and continue**.
  - On the scopes page select **Add or remove scopes**. From the list of scopes select `.../auth/userinfo.email	` and `openid`. Scroll to the bottom of the selected scopes page and click **Update**.
  - Click **Save and continue**.
  - Verify your OAuth consent and click **Back to dashboard**.

  ![OAuth consent summary](../../images/google-setup/connect-users-google-3.png)

5. Navigate to the **APIs and Services > Credentials** dashboard and click **Create credentials > OAuth client ID**.

![OAuth client navigation](../../images/google-setup/connect-users-google-4.png)

  - For **Application type** select `Web application`.
  - For **Name** enter `Infra`.
  - Under **Authorized redirect URIs** select **Add URI** and add `http://localhost:8301`.
  - Click the **Create** button at the bottom of the page.

  ![OAuth credentials create](../../images/google-setup/connect-users-google-5.png)

<details>
  <summary><strong>(Optional) Configure Google for Infra Dashboard Login</strong></summary>

Add a redirect URI: `<your infra host>/login/callback`.

Examples:
  - `https://infra.company.internal/login/callback` (If infra is hosted at `infra.company.internal`)
  - `http://localhost/login/callback` if trying out Infra locally

</details>
<br />

6. Note the **Client ID** and **Client Secret** fields.

![OAuth client details](../../images/google-setup/connect-users-google-6.png)

