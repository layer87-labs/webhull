# Configuration Reference

Webhull uses two config files with distinct roles.

---

## pages.yaml — Build-Time Config

Baked into the container image. Defines site structure, navigation, i18n, SEO, contact form, and UI strings.

### `site`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Site name — used in meta tags and copyright line |
| `baseURL` | string | ✅ | Canonical base URL — used in sitemap, OG tags, hreflang |
| `logoPath` | string | — | Path to header logo (e.g. `/static/img/logo.webp`) |
| `faviconPath` | string | — | Path to favicon (e.g. `/static/img/favicon.svg`) |
| `copyrightStartYear` | int | — | © start year shown in footer |

### `i18n`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `defaultLanguage` | string | ✅ | Fallback language code (e.g. `de`) |
| `languages` | []string | ✅ | All supported language codes — must have matching `content/[lang]/` dirs |

Webhull auto-generates `/de/`, `/en/` routes and hreflang tags.

### `contentDir`

```yaml
contentDir: "content"   # default, relative to site/ root
```

### `navigation`

```yaml
navigation:
  header:
    de:
      - title: "About"
        url: "/about"
        slug: "about"           # must match content filename without .html
        children:               # optional nested items
          - title: "Team"
            url: "/team"
            slug: "team"
  footer:
    de:
      - title: "Company"
        slug: "footer-company"
        url: "#"
        children:
          - title: "Imprint"
            url: "/imprint"
            slug: "imprint"
          - title: "Privacy"
            url: "/privacy"
            slug: "privacy"
```

Active state is set automatically based on current URL.

### `contact`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | — | Activates the contact form endpoint (default: `false`) |
| `recipientName` | string | if enabled | Displayed in confirmation email |
| `recipients` | []string | if enabled | Email addresses that receive form submissions |
| `subject.[lang]` | string | if enabled | Subject template — supports `{{.Name}}` |
| `maxLinks` | int | — | Max URLs allowed in message body (spam filter, default: `2`) |
| `rateLimit.requests` | int | — | Max submissions per window (default: `3`) |
| `rateLimit.window` | duration | — | Rate limit window (default: `15m`) |

### `mail`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `from` | string | if contact | Sender address |
| `fromName` | string | — | Sender display name |
| `templates.[lang].subject` | string | — | Confirmation email subject |
| `templates.[lang].bodyFile` | string | — | Path to HTML template, relative to `site/` |

SMTP credentials live in `config.yaml`, not here.

### `consent`

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Show GDPR cookie banner |
| `categories.[name].required` | bool | Cannot be disabled by user |
| `categories.[name].default` | bool | Enabled by default |
| `i18n.[lang].title` | string | Banner title |
| `i18n.[lang].description` | string | Banner description |
| `i18n.[lang].acceptAll` | string | Accept all button label |
| `i18n.[lang].rejectAll` | string | Reject all button label |
| `i18n.[lang].customize` | string | Customize button label |
| `i18n.[lang].save` | string | Save selection button label |
| `i18n.[lang].categories.[name]` | string | Category label |

Standard categories: `necessary` (always required) + `analytics` (opt-in).

### `analytics`

| Field | Type | Description |
|-------|------|-------------|
| `plausible.enabled` | bool | Enable Plausible integration |
| `plausible.baseURL` | string | Plausible instance URL (self-hosted or cloud) |
| `plausible.domain` | string | Domain to track |
| `plausible.scriptPath` | string | Script variant path (default: `/js/script.js`) |

### `seo`

| Field | Type | Description |
|-------|------|-------------|
| `defaultOGImage` | string | Fallback OG image for social sharing |
| `defaultTwitterCard` | string | `summary` or `summary_large_image` |
| `globalJSONLD` | []string | JSON-LD Schema.org objects applied to all pages |

Webhull auto-generates `/sitemap.xml`, `/robots.txt`, canonical tags, and hreflang.

### `ui.[lang]`

| Field | Type | Description |
|-------|------|-------------|
| `contactURL` | string | Contact link in header CTA |
| `contactLabel` | string | Label for contact link |
| `imprintURL` | string | Imprint page URL |
| `imprintLabel` | string | Imprint link label |
| `privacyURL` | string | Privacy policy URL |
| `privacyLabel` | string | Privacy link label |
| `footerTagline` | string | Tagline shown in footer |
| `allRights` | string | Copyright suffix (e.g. `All rights reserved.`) |
| `contactForm.submitText` | string | Form submit button label |
| `contactForm.fields[].name` | string | Form field identifier |
| `contactForm.fields[].label` | string | Visible label |
| `contactForm.fields[].type` | string | `text`, `email`, `textarea` |
| `contactForm.fields[].required` | bool | Client + server validation |
| `contactForm.fields[].placeholder` | string | Input placeholder |

### Content Frontmatter Fields

Every content file starts with a YAML frontmatter block.

| Field | Required | Description |
|-------|----------|-------------|
| `id` | ✅ | Page identifier — becomes the URL slug |
| `template` | ✅ | `default`, `home`, `contact`, or `legal` |
| `title` | ✅ | Page title used in breadcrumbs |
| `description` | — | `<meta name="description">` |
| `seo_title` | — | Overrides `<title>` tag |
| `seo_description` | — | Overrides meta description |
| `seo_image` | — | OG image override |
| `startPage` | — | `true` → served at `/` (only one page per language) |
| `heroLine1` | home template | Hero heading line 1 |
| `heroLine2` | home template | Hero heading line 2 |
| `heroSubtitle` | home template | Hero subtitle |
| `heroCTA1Text` | home template | Primary CTA button label |
| `heroCTA1Link` | home template | Primary CTA URL |
| `heroCTA2Text` | home template | Secondary CTA button label |
| `heroCTA2Link` | home template | Secondary CTA URL |
| `gate` | — | `true` → page requires HMAC cookie |
| `jsonLD` | — | Inline JSON-LD for this page |

---

## config.yaml — Runtime Config

Mounted at `/app/config/config.yaml` via Helm ConfigMap. Never baked into the image.

Supports `${VAR:default}` env-var expansion throughout.

### `server`

| Field | Default | Description |
|-------|---------|-------------|
| `port` | `8080` | Listen port |
| `host` | `0.0.0.0` | Listen address |
| `environment` | `development` | `development` or `production` |
| `staticDir` | `static` | Path to static files inside `site/` |
| `staticCacheMaxAge` | `8760h` | Browser cache TTL for static assets (1 year) |
| `read_timeout` | `15s` | HTTP read timeout |
| `write_timeout` | `15s` | HTTP write timeout |
| `idle_timeout` | `60s` | Keep-alive timeout |
| `shutdown_timeout` | `30s` | Graceful shutdown window |

### `mail`

| Field | Description |
|-------|-------------|
| `host` | SMTP server hostname |
| `port` | SMTP port (e.g. `465`, `587`) |
| `username` | SMTP username — use `${SMTP_USER}` |
| `password` | SMTP password — use `${SMTP_PASSWORD}` |
| `useTLS` | `true` for TLS/SSL |

### `gate`

| Field | Description |
|-------|-------------|
| `secret` | HMAC secret — use `${GATE_SECRET}`. When set, pages with `gate: true` in frontmatter require a valid HMAC cookie. |

### `health`

| Field | Description |
|-------|-------------|
| `serviceVersion` | Version string returned by health endpoint — use `${APP_VERSION:dev}` |

Health server runs on port `8082` by default.

### Minimal config.yaml

```yaml
server:
  port: "${PORT:8080}"

mail:
  host: "${SMTP_HOST}"
  port: ${SMTP_PORT:587}
  username: "${SMTP_USER}"
  password: "${SMTP_PASSWORD}"
  useTLS: ${SMTP_TLS:true}
```
