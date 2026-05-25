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

Operational settings (server ports, SMTP, analytics credentials, gate secrets) are separated from site structure (navigation, UI strings, content directory path).

```bash
webhull -config deploy/config.yaml -pages site/pages.yaml
```

```
deploy/
  config.yaml      ← operational: server, mail, analytics, gate
                      mounted as a Kubernetes ConfigMap
site/
  pages.yaml       ← site structure: identity, i18n, nav, ui strings
                      baked into the container image
  content/
    de/
      home.html
  static/
```

**Why split?** Operational config contains environment-specific values (ports, credentials) that differ between staging and production. Site structure is stable and belongs in the image. Mounting only `config.yaml` as a ConfigMap means you can update server settings without rebuilding the container.

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
1. Read configPath (required)
2. Expand ${VAR:default} throughout
3. Apply defaults (port=8080, environment=development, etc.)
4. If pagesPath is set: read pagesPath, merge site structure fields
5. If contentDir is set: scan for HTML files, parse frontmatter + body
6. Validate merged config
7. Start server
```

## See also

- [Configuration reference](../reference/configuration.md) — every field, type, and default
- [Environment variables](../reference/environment-variables.md) — all `${VAR}` placeholders
