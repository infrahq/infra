# Configuration

## Overview

### Example

```yaml
providers:
  - name: okta
    okta:
      apiKey: "9101j12-1820j1280-129812908"
  - name: github
    github:
      clientId: 182.180jd0182jd1192
      secretKey: "@kubernetes:infra/githubtoken"
      teams:
        - "developers"
  - name: google
    google:
      clientId: 182.180jd0182jd1192
      clientSecret: "@kubernetes:infra/clientSecret"
      serviceAccount: "@kubernetes:infra/serviceAccount"
      groups:
        - "google"

permissions:
  - provider: okta
    username: "jeff@infrahq.com"
    role: admin
    namespace: default        # optional
```

## Providers

### GitHub

### Google Accounts

### Okta

## Sources

## Destinations

## Secrets
