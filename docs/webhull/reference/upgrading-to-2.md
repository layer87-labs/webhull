---
sidebar_position: 5
title: Upgrading to 2.0
---

# Upgrading to 2.0

webhull 2.0 changes three defaults of the contact form. Version 1.9.2 already contains these changes; upgrade straight to 2.0.0 instead of 1.9.2.

## Trusted proxies

Before 2.0, webhull trusted the `X-Forwarded-For` header of every peer. From 2.0 on, `server.trustedProxies` is empty by default and webhull trusts no proxy: the client IP is the address of the TCP peer.

If webhull runs behind a reverse proxy, ingress controller or load balancer, set `server.trustedProxies` to the address range of that proxy. Otherwise every visitor shares the proxy's address, and the contact form rate limit (3 submissions per 15 minutes by default) applies to all visitors together.

```yaml
server:
  trustedProxies:
    - "192.0.2.0/24"   # the address range of your ingress controller / load balancer
```

List only the proxies directly in front of webhull and make sure, for example with a NetworkPolicy, that nothing else can reach webhull directly. When a forwarding header arrives while `trustedProxies` is empty, webhull logs one warning. Without a proxy in front of webhull (for example local development), leave the list empty.

See [Running behind a reverse proxy](configuration.md#running-behind-a-reverse-proxy).

## Auto-reply templates

The auto-reply goes to the address entered in the form, so it no longer contains submitted values. The only placeholder is `{{CONTACT_ID}}`, the reference number of the submission. Other placeholders such as `{{NAME}}`, `{{MESSAGE}}` or `{{.Name}}` are removed, and webhull logs a warning at startup for each one.

Replace a greeting like `Hello {{NAME}},` with a neutral one and add the reference number if you want it in the mail:

```html
<p>Hello,</p>
<p>Thank you for your message. Your reference: {{CONTACT_ID}}</p>
```

The full submission still goes to `contact.recipients`.

## Contact form limits

| Setting | Default | Effect |
| --- | --- | --- |
| `contact.autoReplyRateLimit` | 20 per hour | Auto-replies across all visitors. Over the cap, the submission still reaches the recipients; only the auto-reply is skipped. |
| `contact.maxBodyBytes` | 32768 | Larger requests get `413`. |
| field `maxLength` | 5000 for `textarea`, 256 otherwise | Longer values get `400`. |

Email values containing line breaks are rejected.

See [Contact form](../features/contact-form.md).
