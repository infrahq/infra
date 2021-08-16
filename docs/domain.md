# Adding a domain

## Find your Infra Server endpoint

```
$ kubectl get svc --namespace infrahq
NAME               TYPE           CLUSTER-IP     EXTERNAL-IP     PORT(S)        AGE
infra-regsitry     LoadBalancer   10.12.11.116   31.58.101.169   80:32326/TCP   1m
```

In this case, Infra is exposed on IP `31.58.101.169`

## Add DNS records

Add the following DNS records to set up automatic LetsEncrypt certificates for your Infra Server.

(Note: replace `infra.example.com` with your domain of choice)

| TYPE         | DOMAIN                           | VALUE                              | Required                      |
| :--------    | :------------------------------  | :--------------------------------- | :---------------              |
| A            | infra.example.com                | 31.58.101.169                      | Yes                           |
| TXT          | infra.example.com                | _acme-challenge.infra.example.com  | Only if behind firewall / VPN |

Note that some Load Balancers (e.g. on AWS) will require using a **CNAME** record instead.

## Login via the new domain

```
infra login infra.example.com
```
