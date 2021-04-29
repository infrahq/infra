# Architecture

## Goals

* Easy to understand and learn
* API-driven
* Single-binary application
* Low-to-no maintenance
* Deploy anywhere (Kubernetes, Docker, Linux, serverless environments)
* Minimal exposure (a signle TCP port)

## Overview

Infra Engine distributes credentials to [Users](#users) or [Machines](#machines) to grant access infrastructure [Destinations](#destinations) such as servers, clusters or databasess.

Users are humans (developers, operators, support team, etc). Machines are servers, containers or applications. Both Users and Machines can be part of [Groups](#groups).

Both Users, Machines and Groups can be provisioned manually. However, it's usually much more convenient to provision Users via **providers** such as Okta, GitHub or Google Accounts.

**Roles** determine if a User or Machine can receive credentials, and what they can do with those credentials.

## Concepts

### Destinations

Destinations are the infrastructure users or machines gain access to. A destination can be a Kubernetes cluster, an SSH host, a database or even the AWS API.

### Users

Users are humans.

### Machines

Non-humans accessing infrastructure, including:
* A VM
* Kubernetes pods
* Lambda functions
* GitHub Actions jobs
* Any other process that cannot log in via SSO.

### Groups

Groups are a set of users or machines that can share access.

### Sources

Sources are sources of identity, and allow a way to synchronize users, groups and machines in bulk to Infra.

Sources do two things:
* Automatically provision users, groups or machines identities in Infra.
* Allow users, groups or machines to prove their identity via single sign on (e.g. logging in with Okta)

### Roles

Roles define which destinations a user, machine or group can access:

* Define fine-grained permissions (destination-dependent)
* Usually map 1-1 to a role for a destination (e.g. Kubernetes, AWS)

### Credentials

Credentials are the destination-specific credentials that are distributed to users & machines.

* Provide access, usually include:
  * Endpoint to reach destination
  * Raw credentials
  * Special convenience formats specific to desination (e.g. KubeConfig for Kubernetes, Database URL)

## Authentication

The authentication to the Infra Engine API is powered via Tokens. Tokens authenticate both **users** and **machines**.

An example token is:

```
sk_Qg4Bo5CWCZuqYqGutmK8LYT2wJ492Ed8zKnDi5YFRSA8
```

## Mesh Mode

Certain destinations support **Mesh Mode**. With **Mesh Mode**, all access is routed through Infra. This is required for many destinations that have little support for identity providers such as Google Kubernetes Engine. Mesh Mode is most useful for User access but can be used with Machines too.

### When to use Mesh Mode

1. The destination does not provide a way to create user credentials
2. The destination does not support identity provider protocols
3. The destination is inaccessible to users or machines (i.e. on premise or in private network)
4. There is a need for advanced auditing logs (behind knowing who logged in where)
5. For security reasons, users or machines cannot have have actual 

### (Coming Soon) Mesh Mode with non TCP requests (e.g. SSH, etc)

## Secrets 

## Security

