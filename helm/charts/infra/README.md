# Infra Chart

A Helm chart for Infra, an infrastructure access management tool for Kubernetes.

The default deployment installs `infra-server` and accompanying services including a frontend service and a PostgreSQL database. The full list of configuration values can be found with `helm show values infrahq/infra`.

## Upgrade

Run the commands below to update the Helm repository and perform the upgrade. Ensure the correct values are specified to avoid any unnecessary changes to the deployment. These commands assume the Helm release is called `infra`.

```
RELEASE_NAME=infra
RELEASE_NAMESPACE=$(helm status $RELEASE_NAME -o json | jq -r .namespace)
HELM_ARGS=  # additional arguments to helm, e.g. --set or --values

helm repo update infrahq
helm -n $RELEASE_NAMESPACE upgrade $RELEASE_NAME infrahq/infra $HELM_ARGS
```

## Rollback

If the upgrade is unsuccessful, the deployment can be rolled back using Helm.

```
RELEASE_NAME=infra
RELEASE_NAMESPACE=$(helm status $RELEASE_NAME -o json | jq -r .namespace)

helm -n $RELEASE_NAMESPACE rollback $RELEASE_NAME
```

The upgrade can be attempted again or you can contact us through GitHub for additional assistance.

## Version 0.20.0 and higher

Infra chart version 0.20.0 introduces two backwards incompatible changes. The first is the migration to PostgreSQL as the default backend database from SQLite. The second is the migration of the database encryption key from within the server PVC to a Kubernetes secrets. This guide will detail the required steps to successfully migrate an Infra deploy to 0.20.0.

### Before Upgrade

#### PostgreSQL Database

This change is automated and should not require manual migration. If your deployment uses an external database, i.e. if your Helm values defines `server.config.dbHost`, you can ignore this section.

#### Database Encryption Key

This change requires manual migration before performing the upgrade. If your deployment uses a database encryption key stored outside of Kubernetes, i.e. if your Helm values defines `server.config.dbEncryptionKey`, you can ignore this section.

The following commands creates a Kubernetes secret with the necessary labels and annotations to masquerade as a Helm-managed resource. It seeds this secrets with the current encryption key such that it’s available to Infra after upgrade. 

These commands assume the Helm release is called `infra`. Update the `RELEASE_NAME` variable if necessary with the correct release name. 

```
RELEASE_NAME=infra
RELEASE_NAMESPACE=$(helm status $RELEASE_NAME -o json | jq -r .namespace)
FULL_NAME=$(kubectl -n $RELEASE_NAMESPACE get deployment -l app.kubernetes.io/instance=$RELEASE_NAME -l app.infrahq.com/component=server -o name | cut -f2 -d/)

ENCRYPTION_KEY=$(mktemp)
kubectl -n $RELEASE_NAMESPACE exec -i deployment/$FULL_NAME -- cat /var/lib/infrahq/server/sqlite3.db.key >$ENCRYPTION_KEY
kubectl -n $RELEASE_NAMESPACE create secret generic $FULL_NAME-encryption-key --from-file=key=$ENCRYPTION_KEY
kubectl -n $RELEASE_NAMESPACE annotate secret $FULL_NAME-encryption-key meta.helm.sh/release-name=$RELEASE_NAME meta.helm.sh/release-namespace=$RELEASE_NAMESPACE 
kubectl -n $RELEASE_NAMESPACE label secret $FULL_NAME-encryption-key app.kubernetes.io/managed-by=Helm
rm $ENCRYPTION_KEY
```

### After Upgrade

Once upgraded, verify the deployment with `infra login` or the Infra UI. Once verified, the server persistent volume claim should be disabled, `server.persistence.enabled=false`, to fully remove the SQLite database and old database encryption key.

### Common Errors

```
$ kubectl logs deployment/infra-server
Defaulted container "server" out of: server, postgres-ready (init)
Error: creating server: db: migration failed: load key: unsealing: opening seal: cipher: message authentication failed
```

This error indicates an issue with the database encryption key migration; the encrypted data has been migrated but the key has not. Rollback and run the database encryption key migration before upgrading.

```
$ kubectl create secret generic infra-server-encryption-key --from-file=key=$ENCRYPTION_KEY
error: failed to create secret secrets "infra-server-encryption-key" already exists
error: --overwrite is false but found the following declared annotation(s): 'meta.helm.sh/release-name' already has a value (infra); 'meta.helm.sh/release-namespace' already has a value (default)
error: 'app.kubernetes.io/managed-by' already has a value (Helm), and --overwrite is false
```

This error indicates that an `infra-server-encryption-key` secret already exists when trying to migrate the encryption key and may be a result of prematurely upgrading the release. Delete the secret and run the encryption key migration again.
