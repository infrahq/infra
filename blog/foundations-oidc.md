---
title: "Foundations of Infra: OIDC"
author: Matt Williams
date: 2022-06-06
---

![Infra on openid](https://raw.githubusercontent.com/infrahq/blog/main/assets/img/OpenID.png)
**You might have seen those letters as you navigate the web, but what do they mean.** Well the first three letters are for OpenID. You may remember this as an early way to login to some of the web apps you used.

![openid](https://raw.githubusercontent.com/infrahq/blog/main/assets/img/InfraOpenID.png)

But versions 1 and 2 of OpenID didn't get a lot of adoption. Version 3, or [OpenID Connect](https://openid.net/connect/), got rid of the XML and custom message signature schema used and instead leverages OAuth 2 which uses TLS. TLS is already in use on every client and server platform today.

So what is OAuth 2? Well OAuth is an authorization framework. Version 2 of OAuth came out around 2013 and simplified the flows quite a bit (while also causing a [little bit](https://www.cnet.com/tech/services-and-software/oauth-2-0-leader-resigns-says-standard-is-bad/) of [Internet drama](https://gist.github.com/nckroy/dd2d4dfc86f7d13045ad715377b6a48f)). OIDC wraps around that and provides authentication. OAuth on its own doesn't really provide a standard method for a user to login and provide a password or other verification step. OIDC is what enables that authentication to happen.

For Infra, we use OIDC providers to validate that you are who you say you are. There are quite a few steps involved with that and in the description below this video I will call out some great resources if you want to understand it in more detail. And we use OIDC and OAuth 2.0 throughout the solution to make sure you always have the right access for you.

Infra is made of a few key components. There is a CLI as well as a web UI. Then there is a server component that stores information in a database. You are going to have one server. Then for each cluster you want to connect to, there is a 'connector'.

First you use the CLI to login. When you do that, it gives you a choice depending on how infra has been configured. You might login locally or with an OIDC provider such as [Okta](https://www.okta.com/) (you may see providers referred to as Identity Providers or IdPs).

Once you have verified your identity, the IdP sends a code back to the Infra CLI. That code, which is just a long string of alphanumeric characters, is then sent to the Infra server. The server then sends that to the IdP which says that that code was just generated for this specific user. The IdP returns some tokens to the server that verify information about the user.

Now we know that you are who you say you are and we use this information to generate an access key that you can then use to access the infrastructure. During the configuration of Infra, you would have been granted access to a number of clusters. You now should see all the clusters you have access to in your kubeconfig files, even if those files were blank before logging in.

![kubeconfig](https://user-images.githubusercontent.com/633681/170083625-79151986-2e96-4ecc-bdcd-ddd5eb5ada74.jpg)

When you use a tool such as kubectl to gain access to one of those clusters, it will check with the server to ensure you still have access. The server generates a JavaScript Web Token, or JWT, though it's usually pronounced as JOT. That JWT is sent over to the cluster and the connector on the cluster sees that the JWT has been signed by the server so it can trust that you really have access.

Assuming you have the access required to perform the action you ran, you will get the corresponding results. We will revalidate that you still have access every 5 minutes. That 5 minute interval is configurable, but if you increase the frequency, you may run into rate limits set by the IdP.

And that is how OIDC works in the context of Infra. This post is also available as a video on our YouTube channel.

{% youtube src="https://www.youtube-nocookie.com/embed/gGPyOUtXoKE" title="What is OIDC and why is it relevant to Infra?" /%}
