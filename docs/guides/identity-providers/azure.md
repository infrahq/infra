# Azure

## Connecting Azure
To connect Azure, run the following command:

```bash
infra providers add azure \
  --url login.microsoftonline.com/${TENANT_ID}/v2.0 \
  --client-id <your azure client id> \
  --client-secret <your azure client secret>
```

## Finding required values

1. Login to the Azure Portal.
2. Navigate to the **Azure Active Directory > App registrations**.

![Register Application](../../images/azure-setup/connect-users-azure-1.png)

3. Click **New registration**.
4. Register the application.
  - For **Name** write **Infra**.
  - For **Redirect URI** select `Web` and add `http://localhost:8301`.
  - Click **Register**.

![Application details](../../images/azure-setup/connect-users-azure-2.png)

5. On the **Overview** tab, click **Client credentials > Add a certificate or secret**.
  - Click **New client secret**
  - In the **Add a client secret** pane select an expiry.
  - **Note** the **client secret value**.

![Add a client secret](../../images/azure-setup/connect-users-azure-3.png)

6. Naviate to **Token configuration**. 
  - Click **Add optional claim**.
  - For **Token type** select **ID**.
  - From the list of claims select the `email` claim.
  - Click **Add**.

![Add the email claim](../../images/azure-setup/connect-users-azure-4.png)

7. From the **Overview** tab copy the **Application (client) ID**, **Directory (tenant) ID**, and **Client Secret** values and provide them into Infra's Dashboard or CLI.

<details>
  <summary><strong>(Optional) Configure Azure for Infra Dashboard Login</strong></summary>


Navigate to the **Authentication** tab and add an additional web redirect URI: `<your infra host>/login/callback`. Examples:
  - `https://infra.company.internal/login/callback` (If infra is hosted at `infra.company.internal`)
  - `http://localhost/login/callback` if trying out Infra locally

</details>
<br />
