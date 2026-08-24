# AGENTS.md — webhull

Guidance for AI coding agents working in this repository.

---

## What this repository is

A **reusable Go binary** that serves multilingual, SEO-optimized websites from YAML config
and HTML content files. Single binary, zero runtime dependencies, container-first.

**This is NOT a CMS.** No admin UI, no page builder, no user management.
It is a developer framework: YAML defines structure, HTML fragments define content,
the binary handles everything else (routing, templating, i18n, forms, analytics, consent,
gate, SEO, minification, cache-busting, health).

The published image (`cr.layer87.de/layer87/webhull`) is the base image that consumer
repos (e.g. `website`) build their container images `FROM`.

---

## Repository layout

```
cmd/
  webhull/
    main.go              # Entrypoint — calls cmd.Run()
    cmd/
      cmd.go             # Flag parsing, config loading, logger wiring, server start

internal/
  app/
    server/
      server.go          # Gin setup, middleware stack, route registration, lifecycle
      handlers.go        # HTTP handlers: page, contact, analytics, consent, gate
    templates/           # templ templates (see Template system below)
      viewmodels.go      # PageData, GatePageData, ConsentBannerData view model structs
      pages/             # Page templates: default, home, contact, legal, notfound
      layout/            # Layout components: header, footer, hero, consent, gate

  pkg/
    config/              # YAML config loading, env-var expansion, split-config merge
      models.go          # SiteConfig, PageConfig, ServerConfig, MailConfig, GateConfig …
      loader.go          # Load(), env expansion ${VAR:default}, path resolution
    content/             # HTML + YAML frontmatter parser → []PageConfig
      loader.go
    i18n/                # Language detection (Cookie → Accept-Language → Default)
    pages/               # Slug index (O(1) lookup), start page resolution
    navigation/          # Header/footer nav with active-state resolution
    forms/               # Validation → spam filter → SMTP dispatch → auto-reply
    consent/             # GDPR cookie consent: read, write, category resolution
    analytics/           # Multi-provider, consent-gated (Plausible + collector)
    seo/                 # Sitemap, robots.txt, meta tags, hreflang, copyright
    security/            # Security headers middleware, BotDetector, rate limiter
    assets/              # Cache-busting: content-hash appended to static asset URLs
    gate/                # Whole-site HMAC cookie gate + /arcon/* gate (independent)
    plugin/              # Declarative data-source plugins: HTTP fetch → allowlist → render → inject
    middleware/          # Cache, Gzip, Logging Gin middleware

deploy/
  Containerfile          # Production image (chainguard/static distroless base)
  config.yaml            # Operational config template (Helm ConfigMap)
  helm/
    Chart.yaml
    values.yaml          # Full Helm values with inline docs

examples/
  multi-page/
    pages.yaml             # Multi-page site config template
    site.yaml              # Legacy monolithic config example
    content/de/            # Example German content pages
    static/                # Example static assets (CSS, JS, images)
  single-page/
    pages.yaml             # Single-page site config template
    content/de/            # Example German single-page content
    content/en/            # Example English single-page content

docs/webhull/
  intro.md               # What webhull is, why it exists
  getting-started/       # Quickstart, configuration model
  content/               # Authoring, templates, SEO
  features/              # Analytics, contact form, access gate
  reference/             # Configuration, env vars, CLI, deployment

.layer87/
  pipeline.yml           # Dagger NER pipeline config (Go build + container + Helm)

.github/workflows/
  build.yml              # PR preview (Go build + test + container)
  release.yml            # Release (relctl → Dagger → cr.layer87.de)
  cleanup.yml            # PR cleanup (delete preview image)

Makefile                 # All dev targets (see Development workflow)
go.mod                   # Module: github.com/Layer87/webhull
renovate.json            # Dependency tracking (cr.layer87.de registry)
```

---

## Architecture

```
cmd.Run()
  │
  ├── config.Load(configPath, pagesPath)
  │     ├── Parse YAML, expand ${VAR:default}
  │     ├── Resolve paths relative to config file location
  │     └── Merge split config (operational + site structure)
  │
  ├── content.Load(contentDir, langs)
  │     └── Parse HTML+frontmatter → append to []PageConfig
  │
  └── server.New(cfg, logger)
        ├── i18n.Service          — language detection middleware, root redirect
        ├── pages.Service         — slug index, start page lookup
        ├── navigation.Service    — header/footer with active state
        ├── consent.Service       — GDPR cookie read/write
        ├── seo.Service           — sitemap, robots.txt, meta tags, hreflang
        ├── security.BotDetector  — UA-based bot detection
        ├── assets.Service        — content-hash cache-busting map
        ├── forms.Service         — validation → spam → SMTP → auto-reply
        ├── analytics.Service     — consent-gated multi-provider dispatch
        ├── gate.Service          — optional: whole-site HMAC cookie gate
        └── gate.Service          — optional: /arcon/* HMAC cookie gate
```

**Request lifecycle:**
1. Middleware stack: Recovery → Gzip → Logging → Cache → Security headers → i18n → Consent
2. Gate middleware (if enabled): check session cookie → redirect `/gate`
3. Handler: slug → O(1) lookup → `buildPageData()` → `renderTemplate()` → ETag → response

---

## Split config model (production)

Web-core accepts two config flags:

```bash
webhull -config deploy/config.yaml -pages site/pages.yaml
```

| Flag | File | Contains | Deployment model |
|---|---|---|---|
| `-config` | `deploy/config.yaml` | `server`, `health`, `mail`, `analytics`, `consent`, `seo`, `gate`, `arconGate` | Helm ConfigMap — differs per environment |
| `-pages` | `pages.yaml` | `site`, `i18n`, `navigation`, `ui`, `contentDir`, `contact`, `pages` | Baked into consumer container image |

Paths (e.g. `contentDir`, `staticDir`, `mail.templates[].bodyFile`) resolve **relative to
the file that defines them**. `staticDir` from `-config` resolves relative to `-config`;
`contentDir` from `-pages` resolves relative to `-pages`.

**Monolithic mode** (`-pages` omitted): both halves in a single file. Still supported —
used in `examples/multi-page/site.yaml`.

**Local development** uses `deploy/config.yaml` + `examples/multi-page/pages.yaml` via `make run`.

---

## Content loading model

Content files live in `{contentDir}/{lang}/{slug}.html`. The slug is the filename without `.html`.

### Frontmatter (YAML between `---` delimiters)

```html
---
id: products             # Unique — same across ALL language variants (required)
template: default        # default | home | contact | legal (required)
title: "Produkte"        # Page title (required)
description: "..."       # Meta description (required)
seo_title: "..."         # Optional: override title in <title> tag
seo_description: "..."   # Optional: override description in meta
seo_priority: 0.8        # Optional: sitemap priority
seo_changefreq: monthly  # Optional: sitemap changefreq
seo_ogimage: "/static/img/og.webp"  # Optional: OG image override
seo_noindex: true        # Optional: exclude from sitemap + robots
heroLine1: "Unsere"      # Optional: hero first line
heroLine2: "Produkte"    # Optional: hero second line (gradient)
heroSubtitle: "..."      # Optional: hero subtitle
---
<!-- HTML content body follows -->
```

### Language routing

Slugs are per-language — no `/de/` prefix. The URL IS the language:
- `/kontakt` → German, `/contact` → English  
- Root `/` detects: Cookie → `Accept-Language` → default → 302 to start page

The **same `id`** must appear in all language variants so the language switcher can
link `/kontakt` ↔ `/contact`. Pages with matching `id` values are grouped automatically.

### Content body formats

Three mutually exclusive formats for the HTML body after the frontmatter block:

| Format | Syntax | Access in template |
|---|---|---|
| No markers | Plain HTML | `data.Content("body")` |
| Named sections (legacy) | `<!-- section: keyname -->` | `data.Content("keyname")` |
| Typed sections (current) | `<!-- section[type,altbg,id=anchor]: Title -->` | `data.Sections` (ordered slice) |

Typed section types: `block`, `grid`, `services`.

---

## Template system

Templates are written with [**templ**](https://github.com/a-h/templ).

- `.templ` source files in `internal/app/templates/`
- Companion `_templ.go` files are **compiled and committed to git** — CI does not run `templ generate`
- **Always run `make generate` (or `make build`) after modifying any `.templ` file** before committing

View model passed to all templates:
```go
type PageData struct {
    Page          *pages.Page
    Meta          seo.MetaTags
    Site          SiteData
    UI            config.UIStringsConfig
    Analytics     AnalyticsData
    Consent       *consent.State
    ConsentConfig *ConsentBannerData
    Assets        *assets.Service
    IsBot         bool
    // ...
}

// Helper methods
data.AssetPath("/static/css/style.css") // → /static/css/style.css?v=88cc6fe6
data.Content("body")                    // → named content section
data.HasContent("hero")                 // → bool
```

---

## Go rules (mandatory)

- **Go ≥ 1.24**
- Idiomatic Go, explicit error handling — no `panic` in application logic
- No global mutable state
- `context.Context` required everywhere
- **All code and comments in English**

### Package rules

| Layer | Path | Rule |
|---|---|---|
| Domain | `internal/pkg/*` | Self-contained, no HTTP logic, no cross-pkg imports |
| App | `internal/app/*` | Server, handlers — may import any `pkg/` |
| Entrypoint | `cmd/*` | Wiring only, no business logic |

No HTTP logic in `pkg/` packages (except `pkg/middleware`).  
No cross-domain imports between `pkg/` packages.

### File conventions

| File | Purpose |
|---|---|
| `models.go` | Domain entities, plain structs |
| `service.go` | Business logic, stateless functions |
| `middleware.go` | Gin middleware functions |
| `manager.go` | High-level orchestration (when needed) |
| `dto.go` | API-facing request/response structures |

### Testing

- Table-driven tests
- Domain packages (`pkg/`) tested in isolation — no external dependencies
- Handlers tested with `net/http/httptest`
- Run: `make test` or `make test-cover`

### Prohibited

- Business logic in handlers
- HTTP logic in `pkg/`
- Global mutable state
- SPA / client-side rendering
- CMS features (admin UI, page builder, user management)
- Framework-specific / hardcoded content for one website

---

## Ports

| Port | Server | Endpoints |
|---|---|---|
| `8080` | Gin HTTP (configurable via `${PORT:8080}`) | all page, API, static routes |
| `8082` | `net/http.Server` (health) | `/health`, `/ready`, `/metrics` |

The health server runs independently of Gin — it stays up even if the main server is busy.

---

## Security

- **Security headers** set on every response via `security.BotDetector` middleware; CSP is Plausible-aware
- **Gate**: whole-site HMAC-SHA256 signed cookie; `ConstantTimeCompare` for timing-attack resistance
- **ArconGate**: independent HMAC gate for `/arcon/*` path only
- **Rate limiter**: 3 attempts / 15 min / IP (in-memory); used by gate login and contact form
- **Gate secret** must be ≥ 32 chars; inject via `${GATE_SECRET}` — never hardcode
- **Plugins**: declarative only, no code execution; `source.headers` values must be
  exactly `${VAR}`/`${VAR:default}` (literal secrets in a manifest are a startup error);
  `select.fields` is a deny-by-default allowlist; render templates use `html/template`
  (auto-escaping); requests never trigger a fetch — only a background loop does. See
  `docs/webhull/features/plugins.md`.

---

## Development workflow

```bash
make dev          # Hot reload via air — fastest iteration loop
make build        # templ generate → go build → build/bin/webhull
make run          # build + run with deploy/config.yaml + examples/multi-page/pages.yaml
make test         # go test ./... -v -race -count=1
make test-cover   # Coverage report → build/coverage/
make audit        # go vet + tests
make generate     # templ generate only (run after editing .templ files)
make container    # Build container image from deploy/Containerfile
```

---

## CI/CD pipeline

| Workflow | Trigger | Action |
|---|---|---|
| `build.yml` | PR → `main` | Go build + audit + test + container → preview image |
| `release.yml` | Push to `main` | relctl creates release → Dagger: build + push + Helm OCI chart |
| `cleanup.yml` | PR closed | Preview image deleted from registry |

**Registry:** `cr.layer87.de/layer87` (container image), `cr.layer87.de/layer87-charts` (Helm chart OCI).  
No other registry is used. `ghcr.io` is not used.

**Pipeline config:** `.layer87/pipeline.yml` — read by the Dagger NER pipeline.  
Includes: Go build config, container config, Helm chart config.

**Self-hosted runner:** `arc-runner-layer87`  
**Reusable workflows:** `Layer87/git-workflows@main`

---

## Tagging & release rules

- **No `v` prefix** — tags are `1.2.3`, never `v1.2.3`
- **Never re-tag** — if a pipeline run fails, fix the root cause and push a new patch tag (e.g. `1.1.1`)
- Tag format follows semver; pre-release tags (`1.2.0-pr-42`) get an exact tag only; stable tags get aliases (`1.2`, `1`, `latest`)

---

## Key constraints

- **`_templ.go` files are committed** — run `make generate` locally after any `.templ` edit
- **No CGO** — `CGO_ENABLED=0`; binary runs in `chainguard/static` (distroless, no shell)
- **Split config is the production model** — `deploy/config.yaml` via Helm ConfigMap, `pages.yaml` baked into consumer image
- **Module path:** `github.com/Layer87/webhull` (public module, private build registry)
- **`examples/` is in the container context** — example content is included in the built image
- **Secrets at runtime only** — SMTP, gate secret, and other credentials via `${VAR}` expansion — never committed
