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

When the consent system is enabled, the Plausible script tag is only rendered after the visitor accepts the `analytics` cookie category. No script is sent before consent.

The gate is applied **server-side**, in the layout template, not inside the script. The Plausible script sends a pageview the moment it loads, so it must not reach the page at all before a decision. Because the rendered HTML therefore depends on the consent cookie, HTML responses carry `Vary: Cookie` and their `ETag` differs per decision.

A consequence: changing the analytics decision requires a page reload to take effect, since the tag is added or removed server-side. `consent.js` triggers that reload automatically whenever a decision flips the analytics category.

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

Whenever client-side tracking is *not* active — no decision yet, `analytics` rejected, JavaScript disabled, or a CLI client — webhull sends an anonymous pageview server-side to the same configured providers. No cookie is set and no personal data is stored, so this needs no consent.

The two paths are mutually exclusive: as soon as `analytics` consent is given, server-side tracking stops and the client script takes over, so pageviews are never counted twice.

## Consent integration

Analytics is gated twice, on purpose. The server does not render either script tag without accepted `analytics` consent, and `/static/js/analytics.js` re-checks the consent cookie before sending any event — so a stale page that was cached with the script in it still stops tracking once consent is withdrawn.

The category key `analytics` is reserved for this: `consent.CategoryAnalytics`. Sites may define any number of additional categories, but this is the one that gates analytics.

See the [Configuration reference](../reference/configuration.md#analytics) for the full `analytics` config block and the [Configuration reference](../reference/configuration.md#consent) for consent category setup.

## Disabling analytics

To run without analytics (default):

```yaml
# simply omit the analytics block, or:
analytics: {}
```

No scripts are injected, no routes are registered.
