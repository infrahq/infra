# Introduction

Infra provides authentication and access management to servers, clusters, and databases.

### Architecture

Infra has three main components:

- **Infra (Centralized API server)**:

  This component of Infra is responsible for authenticating users & services, mapping roles & permissions, indexing the connected infrastructure, and generating **short-lived credentials** for access.

  - [**Authentication**](manage/authentication):

        Infra works by acting as a client to identity providers for authentication. It currently integrates with the identity providers via openID connect (OIDC).

        For local users, Infra acts as an identity provider.

  - [**Access Control**](manage/access-control):

        Infra facilitates access control by mapping users, groups of users, and services by mapping them to roles and permissions in a given system.

        As an example, in the case of Kubernetes, Infra would detect the cluster roles within a Kubernetes cluster, and create the bindings for the users/groups/services when access is given. Infra also supports scoping cluster roles to namespaces.

  - **Short-lived credentials**:

        Infra generates short-lived credentials for users, groups of users, and services to access infrastructure connected via Infra connectors.

        Once a user receives a credential, the credential will be valid for the short period of time to access the infrastructure directly. The credentials will be validated by Infra connectors. This prevents Infra server to become a single point of failure at least for the duration that the credential is valid.

- **Connectors**

      Infra connector is an agent that runs on or within an environment that has access to the infrastructure to be accessed (i.e. within the same VPC).

      For infrastructure that does not support agents such as databases, Infra connector can be installed within the same VPC as long as it has some form of access.

- **Clients**:

      One of the clients available is the Infra CLI. It serves as a way for short-lived credentials to be distributed to client machines.

      Infra's API can be used to help create custom clients for internal CLI tools or used by internal platforms.
