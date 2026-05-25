---
sidebar_position: 1
title: Configuration Reference
---

# Configuration Reference

Complete reference for all fields in `site.yaml` (monolithic) or `config.yaml` + `pages.yaml` (split config).

All string values support `${VAR:default}` environment variable expansion. See [Environment Variables](./environment-variables.md) for a full list.

---

## `site` — Site identity

```yaml
site:
  name: "Acme Corp"
  baseURL: "https://example.com"
  logoPath: "/static/img/logo.webp"
  faviconPath: "/static/img/favicon.webp"
  copyrightStartYear: 2022
  showLangFlags: false
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | — | **Required.** Site name used in `<meta name="author">`, gate page title, and startup log. |
| `baseURL` | string | — | **Required.** Absolute base URL without trailing slash. Used for canonical URLs, OG tags, and sitemap `<loc>`. |
| `logoPath` | string | — | Path to the site logo shown in the header. |
| `faviconPath` | string | — | Path to the favicon (`<link rel="icon">`). |
| `copyrightStartYear` | int | current year | Start year for the copyright range in the footer (e.g. `2022 – 2026`). |
| `showLangFlags` | bool | `false` | Show flag icons in the language switcher. |

---

## `i18n` — Internationalisation

```yaml
i18n:
  defaultLanguage: "en"
  languages: ["en", "de"]
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `defaultLanguage` | string | — | **Required.** Language code for the default language. Used for `hreflang="x-default"` and language detection fallback. |
| `languages` | []string | — | **Required.** All supported language codes. Each language must have a content directory. |

---

## `contentDir` — Content directory

```yaml
contentDir: "content"
```

Path to the directory containing HTML content files, relative to the config file. Structure: `contentDir/{lang}/*.html`. When set, webhull scans the directory at startup and registers a route for every file found.

---

## `navigation` — Header and footer

```yaml
navigation:
  header:
    en:
      - title: "About"
        url: "/about"
        slug: "about"
        children:
          - title: "Team"
            url: "/team"
            slug: "team"
  footer:
    en:
      - title: "Legal"
        url: "#"
        slug: "legal"
        children:
          - title: "Imprint"
            url: "/imprint"
            slug: "imprint"
```

Navigation is defined per language. Each item has `title`, `url`, `slug` (for active-state matching), and optional `children`.

---

## `ui` — Per-language UI strings

```yaml
ui:
  en:
    contactURL: "/contact"
    contactLabel: "Contact"
    footerTagline: "Built with care."
    allRights: "All rights reserved."
    imprintURL: "/imprint"
    imprintLabel: "Imprint"
    privacyURL: "/privacy"
    privacyLabel: "Privacy"
    notFoundTitle: "Page not found"
    notFoundSubtitle: "The page you are looking for does not exist."
    notFoundButton: "Back to home"
    contactForm:
      heading: "Send us a message"
      nameLabel: "Name"
      emailLabel: "Email"
      subjectLabel: "Subject"
      messageLabel: "Message"
      submitText: "Send"
      successMsg: "Thank you! We will be in touch."
      successRefMsg: "Your reference:"
      errorMsg: "Something went wrong. Please try again."
      requiredMsg: "Please fill in all required fields."
```

| Field | Description |
|-------|-------------|
| `contactURL` | URL for the contact CTA button in the header. |
| `contactLabel` | Label for the contact CTA button. |
| `footerTagline` | Tagline text in the footer. |
| `allRights` | "All rights reserved" text in the footer. |
| `imprintURL` / `imprintLabel` | Imprint link in the footer. |
| `privacyURL` / `privacyLabel` | Privacy policy link in the footer. |
| `notFoundTitle` | H1 on the 404 page. |
| `notFoundSubtitle` | Subtitle on the 404 page. |
| `notFoundButton` | Back-to-home button label on the 404 page. |
| `contactForm.*` | Contact form label overrides. All optional — fall back to built-in DE/EN defaults. See [Contact Form](../features/contact-form.md). |

---

## `seo` — SEO defaults

```yaml
seo:
  defaultOGImage: "/static/img/og/default.webp"
  defaultTwitterCard: "summary_large_image"
  globalJsonLD:
    - |
      {
        "@context": "https://schema.org",
        "@type": "Organization",
        "name": "Acme Corp",
        "url": "https://example.com"
      }
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `defaultOGImage` | string | — | Fallback `og:image` for pages without `seo_ogimage`. Relative paths are prefixed with `site.baseURL`. |
| `defaultTwitterCard` | string | `summary_large_image` | Default Twitter Card type. |
| `globalJsonLD` | []string | — | Raw JSON-LD blocks injected on every page. Use for `Organization`, `WebSite` schemas. |

---

## `server` — HTTP server

```yaml
server:
  port: "${PORT:8080}"
  host: "0.0.0.0"
  environment: "${ENVIRONMENT:development}"
  readTimeout: 15s
  writeTimeout: 15s
  idleTimeout: 60s
  shutdownTimeout: 30s
  staticDir: "static"
  cacheMaxAge: 1h
  staticCacheMaxAge: 8760h
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | string | `8080` | HTTP listen port. |
| `host` | string | `0.0.0.0` | HTTP listen address. |
| `environment` | string | `development` | `development` enables Gin debug mode and dev logger. `production` enables release mode and JSON logger. |
| `readTimeout` | duration | `15s` | HTTP read timeout. |
| `writeTimeout` | duration | `15s` | HTTP write timeout. |
| `idleTimeout` | duration | `60s` | HTTP keep-alive idle timeout. |
| `shutdownTimeout` | duration | `30s` | Graceful shutdown deadline on SIGINT/SIGTERM. |
| `staticDir` | string | — | Path to the static files directory. Served at `/static/*`. |
| `cacheMaxAge` | duration | — | `Cache-Control: max-age` for rendered HTML pages. |
| `staticCacheMaxAge` | duration | — | `Cache-Control: max-age` for static assets. Typically set to 1 year (`8760h`) — cache-busting is handled via content-hash query params. |

---

## `health` — Health server

A separate HTTP server for Kubernetes liveness, readiness, and metrics probes. Runs on a different port from the main server so probes are never blocked by application middleware.

```yaml
health:
  enabled: true
  host: "0.0.0.0"
  port: 9090
  healthPath: "/health"
  readyPath: "/ready"
  metricsPath: "/metrics"
  timeout: 5s
  serviceName: "my-site"
  serviceVersion: "1.2.3"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable the health server. |
| `host` | string | `0.0.0.0` | Listen address. |
| `port` | int | — | Listen port. Required when enabled. |
| `healthPath` | string | `/health` | Liveness probe path. Returns `{"status":"ok"}`. |
| `readyPath` | string | `/ready` | Readiness probe path. Returns `{"status":"ready"}`. |
| `metricsPath` | string | `/metrics` | Prometheus-style metrics path. Exposes `webhull_info` and `webhull_uptime_seconds`. |
| `timeout` | duration | — | Read/write/idle timeout for the health server. |
| `serviceName` | string | `webhull` | Value of the `service` label in `webhull_info`. |
| `serviceVersion` | string | `unknown` | Value of the `version` label in `webhull_info`. |

---

## `contact` — Contact form

```yaml
contact:
  enabled: true
  recipientName: "Support"
  recipients:
    - "hello@example.com"
  subject:
    en: "New message from {{.Name}}"
    de: "Neue Nachricht von {{.Name}}"
  maxLinks: 2
  rateLimit:
    requests: 3
    window: 15m
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable the contact form and `/api/contact` endpoint. |
| `recipientName` | string | — | Display name in the `To:` header of outgoing emails. |
| `recipients` | []string | — | Email addresses to deliver form submissions to. |
| `subject` | map[string]string | — | Per-language email subject template. Variables: `{{.Name}}`, `{{.Subject}}`. |
| `maxLinks` | int | `2` | Messages containing more URLs than this are rejected as spam. |
| `rateLimit.requests` | int | `3` | Maximum form submissions per window per IP. |
| `rateLimit.window` | duration | `15m` | Rate limit window. |

See [Contact Form](../features/contact-form.md) for auto-reply templates and label customisation.

---

## `mail` — SMTP

```yaml
mail:
  host: "${SMTP_HOST:smtp.example.com}"
  port: 587
  username: "${SMTP_USERNAME}"
  password: "${SMTP_PASSWORD}"
  from: "noreply@example.com"
  fromName: "Acme Corp"
  useTLS: false
  templates:
    en:
      subject: "We received your message"
      body: "<p>Thank you, {{.Name}}.</p>"
    de:
      subject: "Ihre Nachricht ist angekommen"
      bodyFile: "mail/auto-reply-de.html"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `host` | string | — | SMTP server hostname. |
| `port` | int | `587` | SMTP port. Use `587` for STARTTLS, `465` for TLS-on-connect. |
| `username` | string | — | SMTP authentication username. |
| `password` | string | — | SMTP authentication password. |
| `from` | string | — | Sender email address. |
| `fromName` | string | — | Sender display name. |
| `useTLS` | bool | `false` | Use direct TLS (port 465). `false` = STARTTLS (port 587). |
| `templates` | map | — | Per-language auto-reply templates. `body` = inline HTML; `bodyFile` = path to HTML file (relative to config dir). |

---

## `analytics` — Analytics providers

```yaml
analytics:
  plausible:
    enabled: true
    domain: "example.com"
    baseURL: "https://plausible.io"
    scriptPath: "/js/script.js"
  collector:
    enabled: false
    endpoint: "https://collector.example.com/events"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `plausible.enabled` | bool | `false` | Enable Plausible proxy. |
| `plausible.domain` | string | — | Plausible site domain (`data-domain` attribute). |
| `plausible.baseURL` | string | — | Plausible server base URL. |
| `plausible.scriptPath` | string | `/js/script.js` | Path to the tracking script on the Plausible server. |
| `collector.enabled` | bool | `false` | Enable custom collector proxy. |
| `collector.endpoint` | string | — | HTTP endpoint to forward events to. |

---

## `consent` — GDPR consent banner

```yaml
consent:
  enabled: true
  categories:
    necessary:
      required: true
      default: true
    analytics:
      required: false
      default: false
  i18n:
    en:
      title: "Cookie Settings"
      description: "We use cookies for website operation and analytics."
      acceptAll: "Accept all"
      rejectAll: "Reject all"
      customize: "Customize"
      save: "Save preferences"
      categories:
        necessary: "Necessary"
        analytics: "Analytics"
```

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Show the consent banner on first visit. |
| `categories.<name>.required` | bool | Required categories cannot be toggled off. |
| `categories.<name>.default` | bool | Default state before the visitor decides. |
| `i18n.<lang>.title` | string | Banner title. |
| `i18n.<lang>.description` | string | Banner description. |
| `i18n.<lang>.categories.<name>` | string | Per-category display name in the banner. |

---

## `gate` — Site-wide access gate

```yaml
gate:
  enabled: true
  cookieSecret: "${GATE_SECRET}"
  cookieName: "gate_session"
  cookieMaxAge: 24h
  codes:
    - code: "aurora-pine-7"
      label: "Prospect — Acme Corp"
  msgTooManyAttempts: "Too many attempts. Please try again later."
  msgInvalidCode: "Invalid access code. Please try again."
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Protect the entire site behind a code gate. |
| `cookieSecret` | string | — | **Required when enabled.** HMAC-SHA256 signing key. Min 32 chars. |
| `cookieName` | string | `gate_session` | Session cookie name. |
| `cookieMaxAge` | duration | `24h` | Session cookie lifetime. |
| `codes` | []GateCode | — | **Required when enabled.** List of valid access codes. |
| `codes[].code` | string | — | Plaintext access code. |
| `codes[].label` | string | — | Internal label (never shown to visitors). |
| `msgTooManyAttempts` | string | English default | Error shown after rate limit is exceeded. |
| `msgInvalidCode` | string | English default | Error shown for wrong codes. |

See [Access Gate](../features/access-gate.md) for full documentation.

---

## `arconGate` — Path-scoped access gate

Protects only `/arcon/*`. All other pages remain public.

```yaml
arconGate:
  enabled: true
  contentDir: "arcon"
  cookieSecret: "${ARCON_GATE_SECRET}"
  cookieName: "arcon_session"
  cookieMaxAge: 8h
  title: "My Product – Access"
  codes:
    - code: "cedar-frost-9"
      label: "Demo prospect"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable the arcon gate. |
| `contentDir` | string | — | **Required when enabled.** Directory served at `/arcon/*`. |
| `cookieSecret` | string | — | **Required when enabled.** HMAC-SHA256 signing key. |
| `cookieName` | string | `arcon_session` | Session cookie name. |
| `cookieMaxAge` | duration | `8h` | Session cookie lifetime. |
| `title` | string | `<site.name> – Access` | Gate page browser title. |
| `codes` | []GateCode | — | **Required when enabled.** List of valid access codes. |
