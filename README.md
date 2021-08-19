<p align="center">
  <img src="./docs/images/header.svg" width="838" />
</p>

## Introduction
Infra is **identity and access management** for Kubernetes. Provide any user fine-grained access to Kubernetes clusters via existing identity providers such as Okta, Google Accounts, Azure Active Directory and more.

**Features**:
* One-command access: `infra login`
* No more out of sync Kubeconfig files
* Fine-grained role assignment
* Onboard & offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)

## Quickstart

1. Create `infra.yaml` 
```yaml
# Configure external identity providers
sources:
  - type: okta
    domain: acme.okta.com
    clientId: 0oapn0qwiQPiMIyR35d6
    clientSecret: infra-registry-okta/clientSecret
    apiToken: infra-registry-okta/apiToken

# Map groups (coming soon) or individual users pulled from identity providers
# Roles refer to available roles or cluster-roles currently 
# configured in the cluster. Custom roles are supported. 

users:
  - name: person@example.com
    roles:
      - name: admin
        kind: cluster-role
        clusters:
          - cluster-1
          - cluster-2
```
Please follow [Okta configuration guide](./docs/okta.md) to obtain your Okta API token. 

2. Install Infra Registry with configuration

```
helm repo add infrahq https://helm.infrahq.com
helm repo update

# if you have not yet created the namespace for the deployment add a --create-namespace flag to this command
helm install infra-registry infrahq/registry --namespace infrahq --set-file config=./infra.yaml 
```

3. Connect Kubernetes Cluster(s)

In a web browser visit the Infra Registry dashboard. The URL may be found using: 

```
kubectl get svc -n default -w infra -o jsonpath="{.status.loadBalancer.ingress[*]['ip', 'hostname']}"
```
![Login](https://user-images.githubusercontent.com/251292/128047128-7bb0da64-4111-4116-b39b-03ca70687ad2.png)

Once in the dashboard, navigate to **Infrastructure** and click **Add Cluster**

![Add cluster](https://user-images.githubusercontent.com/251292/128047513-77500f36-b8a7-4b51-afff-f75f63c7fb7d.png)

Run this command to connect an existing Kubernetes cluster. Note, this command can be re-used for multiple clusters or scripted via Infrastructure As Code (IAC).

## Usage Guide 

### Install Infra CLI

**macOS & Linux**

```
brew install infrahq/tap/infra
```

**Windows**

```
scoop bucket add infrahq https://github.com/infrahq/scoop.git
scoop install infra
```

### Login to your Infra Registry

```
infra login <your infra registry endpoint>
```

After login, Infra will automatically synchronize all the Kubernetes clusters configured for the user into their default kubeconfig file. 


### Accessing clusters 

To list all the clusters, please run `infra list`. 

Users can then switch Kubernetes context via `kubectl config use-context <name>` or via any Kubernetes tools. 

## Next Steps 
* [Add a custom domain](./docs/domain.md) to make it easy for sharing with your team 
* [Connect more Kubernetes clusters](./docs/connect.md)
* [Update roles](./docs/permissions.md) 

## Documentation
* [Okta Reference](./docs/okta.md)
* [Helm Chart Reference](./docs/helm.md)
* [CLI Reference](./docs/cli.md)
* [Contributing](./docs/contributing.md)
* [Configuration reference](./docs/configuration.md)

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
