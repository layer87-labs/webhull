---
sidebar_position: 8
---

# Observability

webhull can report two things back to a backend of your choosing: **errors that
happen in the visitor's browser**, and **violations of its own
Content-Security-Policy**.

They are different signals with different purposes that happen to share
infrastructure, and it is worth keeping them apart:

| | Error tracking | CSP reporting |
|---|---|---|
| Answers | why did the page break for this visitor | what did the browser refuse to load |
| A report means | the software has a bug | an attack was stopped, a browser extension interfered, or the policy is too tight |
| Belongs to | development | security, then development |

Error tracking closes a gap that server-side tracing leaves open by
construction. A trace starts when a request arrives; a frontend that crashed
while rendering never sends one. The visitor sees a blank page and the
monitoring sees quiet.

Both targets are configuration. Nothing about them is compiled in — a fixed
endpoint would bind the binary to one installation and turn it into a tool
that reports into someone else's system when run elsewhere.

## Configuration

```yaml
observability:
  errorTracking:
    enabled: true
    # Sentry-format DSN. Public by design: it ends up in every delivered page
    # and is good for writing and nothing else.
    dsn: "${WEBHULL_ERROR_DSN}"
    # Where the browser loads the SDK from. There is no default on purpose —
    # see "Where the SDK comes from" below.
    sdkURL: "/static/js/sentry.min.js"
    environment: "production"   # falls back to server.environment
    release: "${IMAGE_TAG:}"
    sampleRate: 1.0

  csp:
    # A relative path is the right choice; see "Why the report path is
    # relative" below.
    reportURI: "/csp?glitchtip_key=${WEBHULL_CSP_KEY}"
    # true sends Content-Security-Policy-Report-Only instead of enforcing.
    reportOnly: false
```

Every value goes through webhull's `${VAR}` and `${VAR:default}` expansion, so
secrets and per-environment values stay out of the config file.

If `enabled` is true but `dsn` or `sdkURL` is missing, error tracking stays off
and the site serves pages as usual. A missing error tracker is a gap in
observability, not a reason to take a site down.

## Where the SDK comes from

`sdkURL` has no default, and that is deliberate. Shipping one would decide for
every operator where their visitors' browsers fetch code from — a third-party
CDN that sees every page view, on every site running webhull.

Two sensible answers:

- **Serve it yourself.** Drop the SDK bundle into your `staticDir` as
  `js/sentry.min.js` and point `sdkURL` at `/static/js/sentry.min.js`. Nothing
  leaves your origin, and the default CSP already allows it.
- **Name a CDN deliberately.** webhull adds its origin to `script-src`
  automatically, so the policy will not block what you chose.

## Why the report path is relative

`report-uri` predates the Reporting API and fires a POST with no preflight, so
a cross-origin endpoint works today. But `report-uri` is deprecated, and its
successor — `Reporting-Endpoints` with `report-to` — requires a CORS preflight
for cross-origin endpoints. A central reporting address is a solution that
breaks exactly when you migrate.

A relative path resolves same-origin and stays preflight-free through both
mechanisms. Terminate it at your reverse proxy and forward from there.

**One caveat that comes with same-origin:** cross-origin reporting endpoints
never receive credentials — the Reporting API blocks that deliberately. A
same-origin endpoint does receive them, so every violation report carries the
visitor's cookies. **Strip the `Cookie` header at the proxy** before forwarding
the report on. Without that line, the safer-looking design leaks sessions.

## What is not collected

The bundled initialiser configures the SDK to send no IP address, no cookies
and no request headers (`sendDefaultPii: false`), and disables performance
tracing and session replay. This channel exists to catch exceptions, not to
become a second analytics product.

Reports whose stack frames live entirely inside a browser extension are dropped
before sending. Extensions are the largest single source of noise in any
real-world deployment, and they say nothing about your site.

## Consent

Error tracking is **not** gated behind analytics consent, unlike the Plausible
integration. The two do different things: analytics observes the visitor, error
tracking observes the software. Configured as above it collects nothing about
the person — and a channel that only fires after consent would miss exactly the
visitors whose browser broke before the banner loaded.

If your legal assessment differs, set `enabled: false` and inject your own
snippet through a plugin.

## HSTS

Related, because it is set by the same middleware: `Strict-Transport-Security`
is sent in production as `max-age=31536000`, without `includeSubDomains` and
without `preload`.

Both are available:

```yaml
server:
  hsts:
    includeSubdomains: true
    preload: true
```

Both default to false, and that default is the point. `includeSubDomains`
applies to **every** subdomain, including future ones and ones hosted
elsewhere; `preload` enters the domain into a list shipped inside browsers,
where removal takes months. Neither belongs in a default, and neither should be
switched on without knowing every hostname under the domain.
