---
title: What is Infra?
position: 1
---

# What is Infra?

## Introduction

Infra is a service for managing secure access to infrastructure such as Kubernetes. It integrates with existing identity providers such as Okta to automatically grant & revoke access to the right users and groups at the right time.

### Features

- **Discover & access** infrastructure via a single command: `infra login`
- **No more out-of-sync credentials** for users (e.g. Kubeconfig)
- **Okta, Google, Azure AD** identity provider support for onboarding and offboarding
- **Fine-grained** access to specific resources that works with existing RBAC rules
- **API-first design** for managing access as code or via existing tooling
- [**Open-source**](https://github.com/infrahq/infra) and can be deployed anywhere

### Coming Soon

- **Dynamic access** to coordinate access with systems like PagerDuty
- **Access requests** to eliminate static access
- **Audit logs** for who did what, when

### Walkthrough

{% youtube id="kxlIDUPu-AY" /%}

## Use Cases

### Automatic onboarding & offboarding

Infra includes deep integration with identity providers such as [Okta](../manage/idp/okta.md). Users are automatically onboarded and provided access to the resources they need without having to create additional accounts. Unlike other tooling, Infra continously verifies users' information with upstream identity providers so access is revoked immediately.

### Configure access as code

Infra supports configuring **access as code** via Git-managed configuration.

Identity providers, users, groups and more can be entirely defined in code, meaning all access is traced back into GitHub, GitLab or other source control systems. For more about this see [Configuring Users in Helm Reference](../reference/helm.md#Users)

### Fine-grained access

Most RBAC or access tooling is _coarse_ grained, meaning users usually receive **admin or nothing** access to infrastructure. With Infra, users or groups can be provided access to specific resources. For more about this see [Working with Roles](../manage/roles.md)

### Dynamic or just-in-time access

Using Infra's API, access can be granted and revoked on-the-fly for users who need it for a limited amount of time. For example, if `suzie@infrahq.com` is starting their on-call schedule, they can be granted and revoked access automatically via an `infra` CLI command or API call. For more about this see [Dynamic Access](../using/dynamic.md)

### Multi-cloud access

Infra works on any major cloud provider and doesn't depend on any existing identity & access management system such as AWS, Google Cloud or Azure IAM. For more about this see [Multi Cloud Access](../using/multicloud.md)
