---
name: webhull
description: Build production-ready websites with Webhull, a config-driven SSR framework (Go, Templ, Gin). Use when building static sites, landing pages, portfolios, or B2B sites that need multilingual support, contact forms, SEO optimization, GDPR-compliant cookie consent, and container deployment. Not suitable for SPAs, real-time apps, or database-driven content.
---

# webhull

Use this skill to build complete, deployable websites with Webhull — config-driven, stateless, single binary.

## Framework Architecture

- **Single binary** — Go SSR, Templ + Gin, zero Node/npm runtime
- **Two config files** — `site/pages.yaml` (baked into image) + `site/config.yaml` (mounted at runtime via Helm ConfigMap)
- **Content** — HTML fragments with YAML frontmatter, one file per page per language
- **Container image** — `FROM ghcr.io/layer87-labs/webhull:1.2.1`

## Project Layout

```
my-site/
├── Containerfile                   # FROM ghcr.io/layer87-labs/webhull:1.2.1 + COPY site
├── docker-compose.yml              # Local dev (webhull + mailpit)
├── site/
│   ├── pages.yaml                  # Site structure — baked in image
│   ├── config.yaml                 # Runtime config — mounted via Helm ConfigMap
│   ├── content/
│   │   └── de/                     # One dir per language (de, en, fr, …)
│   │       ├── home.html           # startPage: true
│   │       ├── contact.html        # template: contact
│   │       ├── imprint.html        # template: legal
│   │       └── privacy.html        # template: legal
│   ├── mail/
│   │   └── de-confirmation.html    # Contact form confirmation email
│   └── static/
│       ├── css/style.css
│       ├── fonts/                  # Self-hosted WOFF2 — no Google Fonts
│       ├── js/
│       └── img/
```

## Quickstart

```bash
# 1 — Create structure
mkdir -p site/{content/de,mail,static/{css,fonts,js,img}}

# 2 — Containerfile
echo 'FROM ghcr.io/layer87-labs/webhull:1.2.1
COPY site /app/site' > Containerfile

# 3 — docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: "3.8"
services:
  webhull:
    build: .
    ports: ["8080:8080"]
    volumes: ["./site:/app/site"]
    environment:
      PORT: "8080"
      SMTP_HOST: "mailpit"
      SMTP_PORT: "1025"
      SMTP_TLS: "false"
  mailpit:
    image: axllent/mailpit:latest
    ports: ["8025:8025", "1025:1025"]
EOF

# 4 — Add pages.yaml, config.yaml, content files (see references/)

# 5 — Run
docker-compose up --build
# → http://localhost:8080
# → Mailpit: http://localhost:8025
```

## Content Files

All content files are **HTML fragments** — no `<html>`, `<head>`, `<body>`, `<nav>`, or `<footer>` tags. Webhull renders the full shell around them.

```html
---
id: home
template: home
title: "Home"
seo_title: "Company — Tagline"
seo_description: "Short meta description."
heroLine1: "Line One"
heroLine2: "Line Two"
heroSubtitle: "Subtitle text."
heroCTA1Text: "Get Started"
heroCTA1Link: "/contact"
heroCTA2Text: "Learn More"
heroCTA2Link: "/about"
startPage: true
---

<section id="about">
  <h2>About Us</h2>
  <p>Content here.</p>
</section>
```

**Available templates:** `default` · `home` · `contact` · `legal`

## Rules

- Never edit Webhull framework source — content and config only
- `config.yaml` must NOT be baked into the container image — runtime mount only
- All secrets via env vars using `${VAR:default}` syntax
- Self-hosted fonts only — no Google Fonts, no external CDN references
- No frontend frameworks (React, Vue) — vanilla JS only
- No database — Webhull is stateless
- Every content file must have a YAML frontmatter block with `id`, `template`, `title`
- Navigation slugs must match content file names (without `.html`)

## Enforcement Gate

Before submitting, verify:

- [ ] `pages.yaml` has `site.name`, `site.baseURL`, `i18n.defaultLanguage`, `i18n.languages`
- [ ] `config.yaml` contains no hardcoded secrets
- [ ] Content files are HTML fragments (no `<html>` / `<body>` tags)
- [ ] Each content file has frontmatter with `id`, `template`, `title`
- [ ] Navigation slugs match content filenames
- [ ] `Containerfile` uses `ghcr.io/layer87-labs/webhull:1.2.1`
- [ ] Contact form: `contact.enabled: true` + `mail.templates` + `mail/de-confirmation.html` exist
- [ ] `docker-compose up --build` runs without errors

If any item is unchecked — stop and fix before continuing.

## Anti-Patterns

- Putting `config.yaml` in the Docker image (`COPY site /app/site` copies it — keep secrets out of `config.yaml` or exclude via `.dockerignore`)
- Creating `<nav>` or `<footer>` tags inside content files — Webhull renders these automatically
- Using template names other than `default`, `home`, `contact`, `legal`
- Hardcoding SMTP credentials in `config.yaml` instead of `${SMTP_PASSWORD}`
- Using Google Fonts or any CDN-hosted assets

## References

- [Full Configuration Reference](references/CONFIG.md) — all `pages.yaml` + `config.yaml` fields
- [Copy-Paste Patterns](references/PATTERNS.md) — four real-world site templates
- [Agent Workflow](references/WORKFLOW.md) — step-by-step from empty dir to deploy
