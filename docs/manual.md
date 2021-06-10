# Manually manage users

## Configure Infra

Configure Infra via `infra.yaml` with a single admin user:

```yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: infra
  namespace: infra
data:
  infra.yaml: |
    grants:
      - user: admin@example.com
        role: admin
EOF
```

## Create a user and log in

Create the user and login token via `kubectl`:

```
$ kubectl -n infra exec infra-0 -- infra user create admin@example.com passw0rd
admin@example.com
```

Finally, log in as `admin@example.com` with the token created in the previous step:

```
$ infra login infra.example.com
? Email admin@example.com
? Password **********
✔ Logging in with username & password... success
✔ Logged in...
✔ Kubeconfig updated
```

That's it. You now have cluster access as `admin@example.com` with with `admin` role.

```
$ kubectl get pods -A
kube-system   coredns-56b458df85-7z4ds          1/1     Running   0          2d4h
kube-system   coredns-56b458df85-wx48l          1/1     Running   0          2d4h
kube-system   kube-proxy-cxn9c                  1/1     Running   0          2d4h
kube-system   kube-proxy-nmnpb                  1/1     Running   0          2d4h
kube-system   metrics-server-5fbdc54f8c-nf85v   1/1     Running   0          46h
```