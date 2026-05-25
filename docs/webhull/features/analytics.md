---
sidebar_position: 1
title: Analytics
---

# Analytics

webhull supports two analytics providers simultaneously: **Plausible Analytics** (script-based, privacy-friendly) and a **custom collector** (HTTP event proxy). Both are optional, both are consent-gated.

## Plausible Analytics

Plausible events are proxied through webhull — visitors never connect directly to the Plausible server. This avoids ad-blockers and keeps all traffic first-party.

### Configuration

```yaml
analytics:
  plausible:
    enabled: true
    domain: "example.com"              # your Plausible site domain
    baseURL: "https://plausible.io"    # or your self-hosted instance
    scriptPath: "/js/script.js"        # path on the Plausible server
```

### How the proxy works

webhull registers two routes:

| Route | Function |
|-------|----------|
| `GET /js/script.js` | Fetches the Plausible tracking script from `baseURL/scriptPath` and serves it with a 30-minute cache. |
| `POST /api/event` | Forwards pageview events to `baseURL/api/event`, preserving the visitor's IP and `User-Agent` for accurate analytics. |

The script tag injected in the page `<head>`:

```html
<script defer
  data-domain="example.com"
  data-api="/api/event"
  src="/js/script.js">
</script>
```

### Consent gating

When the consent system is enabled, the Plausible script is only injected after the visitor accepts the `analytics` cookie category. No script is sent before consent.

## Custom collector

The collector forwards analytics events to any HTTP endpoint — useful for self-built event pipelines or BI tools.

```yaml
analytics:
  collector:
    enabled: true
    endpoint: "https://collector.example.com/events"
```

webhull injects `/static/js/analytics.js` which POSTs structured events (pageview, outbound link click, etc.) to `/api/collect`. The server forwards these to `endpoint`.

Like Plausible, the collector script is also consent-gated.

## Server-side tracking

webhull performs server-side pageview tracking as a fallback when client-side JavaScript is inactive (bots, CLI tools, JS-disabled browsers). These server-side events are forwarded to the same configured providers without any client involvement.

## Consent integration

The analytics scripts (`/static/js/analytics.js`) check the consent cookie before sending any events. If the visitor has not decided or has rejected the `analytics` category, no events are sent.

See the [Configuration reference](../reference/configuration.md#analytics) for the full `analytics` config block and the [Configuration reference](../reference/configuration.md#consent) for consent category setup.

## Disabling analytics

To run without analytics (default):

```yaml
# simply omit the analytics block, or:
analytics: {}
```

No scripts are injected, no routes are registered.
