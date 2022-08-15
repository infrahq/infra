---
title: "Foundations of Infra: JWT"
author: Matt Williams
date: 2022-05-31
---

![Infra on jwt](https://raw.githubusercontent.com/infrahq/blog/main/assets/img/JWTHero.jpg)

**JWT. JSON Web Token.** But it's usually pronounced as jot. If you hear someone say something about a jot, it might actually be about a JSON Web Token. What is a JSON Web Token? It's simply a way to share some fact between services in a verifiable way.

The actual RFC for JWT says that it is a compact, url-safe means of representing claims to be transferred between two parties and that the claim can be cryptographically signed. A claim is any 'piece of information' about a subject. URL-Safe just means using any characters that are allowed in any URI. And cryptographically signed means that the payload cannot be modified without invalidating the signature.

![Infra on jwt](https://raw.githubusercontent.com/infrahq/blog/main/assets/img/jwtspec.jpg)

If you look around the web for information about JWT, you will find a lot of articles and videos that explain how to use them, along with a bunch that say they are terrible and no one should ever use them. The folks who say they are terrible, are mostly focused on one use-case, and two implementation details.

The use case they are against is that of browser to service authentication, and mostly because, by default, the data in a JWT is unencrypted and can't be revoked before the expiration date defined in the token. You can encrypt the data and set short expiration times, but that complicates things and by the time you do all that, there are other solutions such as session tokens that may be more appropriate.

Infra uses JWT for a different purpose and sets a short expiration time. In [the video about OIDC](2022-06-06-foundations-of-infra.md), you saw that we use JWT to record the claim of what this user's email address is and what groups they belong to. That on its own does not guarantee access to resources since the destination determines what access that user or group should have at run time.

The cryptographic signature on the claim allows the client and the destination to be confident that the user is who they claim to be and the groups are the groups they do in fact belong to. If the payload was modified, the signature would be then be invalid and the request would be rejected. Let's take a look at one of these tokens. First the raw JWT looks like this.

```
eyJhbGciOiJFZERTQSIsImtpZCI6IkpPdjJISmxLYjBKTWNfQVRy
R0JLbWFZNmdRTjdlUU5RWXh0ZXlGaHJDTEE9IiwidHlwIjoiSldU
In0.eyJleHAiOjE2NTQ4ODU0NDcsImdyb3VwcyI6bnVsbCwiaWF0
IjoxNjU0ODg1MTQ3LCJuYW1lIjoibWF0dEBpbmZyYWhxLmNvbSI
sIm5iZiI6MTY1NDg4NDg0Nywibm9uY2UiOiJPeFFwTUZTdjRwIn0.
2NSqYsFTa7ILrQljqKOa5Sx51lcPisITkQGtCCSaesf_KJZHotAB
aIARPPvBiu4bQuGPxOOxA8wAYArcf4qiCQ
```

A JWT is made of three components. First there is the header, then the payload, and finally the signature. And those three components are separated using dots or periods. So lets pull out the middle component. Each of these components is Base64 encoded. That's what ensures the URL safe part that we mentioned before. Base64 encoding is not encryption, just a standard way of encoding the text.

Now that that component is Base64 decoded, we see that the payload is this. (If you would like to try this yourself, visit [jwt.io](https://jwt.io) and paste the encoded text above into the Encoded text block.)

```
{
  "exp": 1654885447,
  "groups": null,
  "iat": 1654885147,
  "name": "matt@infrahq.com",
  "nbf": 1654884847,
  "nonce": "OxQpMFSv4p"
}
```

The **iat** value is the time this token was issued. **nbf** says that this token is not valid before this time. And **exp** is the time when this token expires. We can translate those Unix Timestamps to more common date and time formats and can see that this token has a lifetime of 5 minutes. There is also a **nonce** value in there that ensures the request is unique.

So the last two values in the token are the actual claim. Groups and Name. The **name** is the user who is running the command and the **groups** lists out the groups this user is a member of.

That information is used by the destination to determine what roles this user should be allowed to assume.

And that is what a JWT is and how they are used in Infra.

If you would like to see a video version of this document, check out this from our YouTube channel.

{% youtube src="https://www.youtube-nocookie.com/embed/UuKXMDHzcPc" title="Infra Fundamentals: JSON Web Tokens" /%}
