# Okta


```bash
infra providers add Okta \
  --url <your okta url (or domain)> \
  --client-id <your okta client id> \
  --client-secret <your okta client secret>
```
[block:callout]
{
  "type": "danger",
  "body": "The Okta client secret is sensitive information which should not be stored in the Infra configuration file. In order for Infra to access this secret values it should be stored in a secret provider."
}
[/block]
<br/>

**To find the above values:**

1. Login to the Okta dashboard.
2. Under the left menu click **Applications > Applications**.  
Click **Create App Integration**.  
Select **OIDC – OpenID Connect** and **Web Application**.  
Click **Next**.


[block:image]
{
  "images": [
    {
      "image": [
        "https://files.readme.io/4c7e05b-okta1.png",
        "okta1.png",
        1086,
        802,
        "#eceef2"
      ]
    }
  ]
}
[/block]
3. For **App integration name** write **Infra**.  
In **General Settings** > **Grant type** select **Authorization Code** and **Refresh Token**.  
For **Sign-in redirect URIs** write `http://localhost:8301`. For **Assignments** select the groups which will have access through Infra.  
Click **Save**.

[block:image]
{
  "images": [
    {
      "image": [
        "https://files.readme.io/fac622f-okta2.png",
        "okta2.png",
        1086,
        1040,
        "#ebeced"
      ]
    }
  ]
}
[/block]
4. On the **General** tab, **note** the **Client ID**, **Client Secret**, and **Okta domain** for adding your Okta information to Infra later.


[block:image]
{
  "images": [
    {
      "image": [
        "https://files.readme.io/e8f39b1-okta4.png",
        "okta4.png",
        1098,
        1769,
        "#eff0f1"
      ]
    }
  ]
}
[/block]
5. While still on the screen for the application you just created navigate to the **Sign On** tab.  
On the **OpenID Connect ID Token** select **Edit**.  
Update the **Groups claim filter** to `groups` `Matches regex` `.*`.  
Click **Save**.


[block:image]
{
  "images": [
    {
      "image": [
        "https://files.readme.io/ddd4a74-okta5.png",
        "okta5.png",
        1022,
        1129,
        "#eef1f8"
      ]
    }
  ]
}
[/block]

[block:image]
{
  "images": [
    {
      "image": []
    }
  ]
}
[/block]
