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
    permissions:
      - user: example@acme.com
        permission: view
EOF
```

## Create a user and log in

Create the user and login token via `kubectl`:

```
$ kubectl -n infra exec infra-0 -- infra user create example@acme.com
usr_js08jsec08

$ kubectl -n infra exec infra-0 -- infra token create usr_js08jsec08
sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7
```

Finally, log in as `example@acme.com` with the token created in the previous step:

```
$ infra login --token sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7 infra.acme.com
✔ Logging in with Token... success
✔ Logged in as example@acme.com
✔ Kubeconfig updated
```

That's it. You now have cluster access as `example@acme.com` with read-only `view` permissions.

```
$ kubectl get pods -A
kube-system   coredns-56b458df85-7z4ds          1/1     Running   0          2d4h
kube-system   coredns-56b458df85-wx48l          1/1     Running   0          2d4h
kube-system   kube-proxy-cxn9c                  1/1     Running   0          2d4h
kube-system   kube-proxy-nmnpb                  1/1     Running   0          2d4h
kube-system   metrics-server-5fbdc54f8c-nf85v   1/1     Running   0          46h

$ kubectl delete -n kube-system pod/kube-proxy-cxn9c # permission denied
```