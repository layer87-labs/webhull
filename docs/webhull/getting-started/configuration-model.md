---
sidebar_position: 2
title: Configuration Model
---

# Configuration Model

webhull supports two configuration shapes. Both are YAML. Understanding which to use is the first architectural decision when setting up a new site.

## Monolithic config (simple)

Everything — operational settings, site structure, navigation, content directory — lives in a single `site.yaml`. This is the simplest setup and works well for most sites.

```bash
webhull -config site.yaml
```

```
site.yaml
content/
  en/
    home.html
    about.html
static/
```

## Split config (production)

Operational settings (server ports, SMTP credentials, health probes) are separated from site structure (navigation, UI strings, content directory path, consent, analytics, SEO, mail identity).

```bash
webhull -config deploy/config.yaml -pages site/pages.yaml
```

```
deploy/
  config.yaml      ← operational: server, health, mail credentials, contact rate limits
                      mounted as a Kubernetes ConfigMap
site/
  pages.yaml       ← site structure: identity, i18n, nav, ui strings, consent,
                      analytics, seo, mail identity (from/fromName/templates)
                      baked into the container image
  content/
    de/
      home.html
  static/
```

**Why split?** Operational config contains environment-specific values (ports, credentials) that differ between staging and production. Site structure is stable and belongs in the image. Mounting only `config.yaml` as a ConfigMap means you can update server settings without rebuilding the container.

### What goes where?

| Category | `config.yaml` (ops) | `pages.yaml` (site) |
|----------|---------------------|---------------------|
| Server (port, host, timeouts) | override | — |
| Health probes | override | — |
| SMTP connection (host, port, username, password, useTLS) | ✅ | — |
| Mail identity (from, fromName, templates) | override | ✅ |
| Contact form enabled | — | ✅ |
| Contact operational params (maxLinks, rateLimit) | override | — |
| Contact recipients & subject | — | ✅ |
| Analytics (plausible, collector) | — | ✅ |
| Consent (enabled, categories, i18n) | — | ✅ |
| SEO defaults (ogImage, twitterCard, jsonLD) | — | ✅ |
| Site identity (name, baseURL, logo) | — | ✅ |
| Navigation | — | ✅ |
| UI strings | — | ✅ |
| Gate secrets | ✅ | — |

**"override"** = has sane defaults; only set in ops config if you need to change them.

### Auto-derived fields (DRY)

These fields are auto-derived from `site.baseURL` when not explicitly set:

| Field | Derived from | Example |
|-------|-------------|---------|
| `analytics.plausible.domain` | `site.baseURL` hostname | `studio-optimays.de` |
| `mail.from` | `noreply@` + hostname | `noreply@studio-optimays.de` |
| `mail.fromName` | `site.name` | `Studio OptiMayS` |
| `health.serviceName` | hostname (dots → dashes) | `studio-optimays-de` |

All auto-derived values can be overridden in either config file.

Both files support `${VAR:default}` expansion — so even values in `pages.yaml` can reference env vars (e.g. `${PLAUSIBLE_BASE_URL:https://...}`).

When a field is defined in **both** files, `pages.yaml` takes precedence (it's merged on top of `config.yaml`). This ensures the image is the single source of truth for site behaviour.

## Environment variable expansion

Every string value in YAML supports `${VAR:default}` expansion:

```yaml
site:
  baseURL: "${BASE_URL:http://localhost:8080}"

mail:
  password: "${SMTP_PASSWORD}"      # no default — fails loudly if unset
  host: "${SMTP_HOST:smtp.example.com}"
```

Rules:
- `${VAR}` — expanded from env; empty string if unset (same as `os.Getenv`)
- `${VAR:default}` — expanded from env; falls back to `default` if unset or empty
- Standard `$VAR` syntax is also supported (no default possible)

Expansion happens at load time, before YAML parsing — the binary never sees the raw variable names in production.

## Startup validation

webhull validates the config before starting. Common errors caught at startup:

| Error | Cause |
|-------|-------|
| `site.name is required` | `site.name` is empty |
| `site.baseURL is required` | `site.baseURL` is empty |
| `page "X" missing i18n for language "Y"` | Content file missing for a configured language |
| `duplicate slug "X" for language "Y"` | Two pages share the same URL slug |
| `gate.cookieSecret is required` | Gate enabled but no secret provided |

Validate config without starting the server:

```bash
webhull -config site.yaml -validate
# config OK — site="My Site" languages=[en] pages=5
```

## Config loading order

```
1. Read configPath (required) — parse YAML, expand ${VAR:default}
2. If pagesPath is set: read pagesPath, expand, merge onto config
   (pages.yaml wins for: site, i18n, nav, ui, consent, analytics, seo,
    contact.enabled, recipients, subject, mail identity)
3. Apply defaults (fills zero-value fields with sane production defaults)
4. Auto-derive domain-dependent fields from site.baseURL
5. Resolve file paths (contentDir, staticDir, mail templates)
6. Validate merged config
7. Start server
```

## See also

- [Configuration reference](../reference/configuration.md) — every field, type, and default
- [Environment variables](../reference/environment-variables.md) — all `${VAR}` placeholders
