# Quickstart

## Install Infra CLI

```bash
# macOS
$ curl --url "https://github.com/infrahq/infra/releases/download/latest/infra-darwin-$(uname -m)" --output /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Linux
$ curl --url "https://github.com/infrahq/infra/releases/download/latest/infra-linux-$(uname -m)" --output /usr/local/bin/infra && chmod +x /usr/local/bin/infra

# Windows 10
$ curl.exe --url "https://github.com/infrahq/infra/releases/download/latest/infra-windows-amd64.exe" --output infra.exe
```

## Deploy Infra Engine on Kubernetes

```
$ kubectl apply -f https://raw.githubusercontent.com/infrahq/infra/master/deploy/kubernetes.yaml
```

Find the endpoint on which Infra Engine is exposed:

```
$ kubectl get svc --namespace infra
NAME      TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra     LoadBalancer   10.12.11.116   31.58.101.169   80:32326/TCP   1m
```

Use a DNS provider (Route53, Cloudflare, etc.) to map a domain (e.g. `infra.acme.com`) to the external IP (e.g. `31.58.101.169`).

## Create your first user

```
$ kubectl exec -n infra infra-0 -- infra users create admin@acme.com --permission view

User admin@acme.com added. Please share the following command with them so they can log in:

infra login --token sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7
```

## Log in

```
$ infra login --token sk_r6Khd35Dt3Q4KgyuPFw2NkRkGpgorI8uyDgpW215quR7 infra.acme.com
✔ Logging in with Token... success
✔ Logged in as admin@acme.com
✔ Kubeconfig updated

$ kubectl get pods
...
$ kubectl -n infra delete pod/infra-0 # will fail because you have view permissions
```

## List users

```
$ infra users ls
USER ID             PROVIDERS   EMAIL               CREATED         PERMISSION 
usr_bja9d8h3971s    token       admin@acme.com      1 minute ago    view
```
