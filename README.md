# webhull

A fast, config-driven web server for multilingual, SEO-optimised websites. Single binary, zero runtime dependencies.

[![CI](https://github.com/layer87-labs/webhull/actions/workflows/ci.yml/badge.svg)](https://github.com/layer87-labs/webhull/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/layer87-labs/webhull)](https://goreportcard.com/report/github.com/layer87-labs/webhull)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## What it is

webhull serves complete, production-ready websites from YAML configuration and HTML content files. No database, no admin UI, no CMS runtime. One binary, one config file.

- **Multilingual** — i18n built in, `hreflang` generated automatically
- **SEO-first** — sitemap, canonical URLs, Open Graph, JSON-LD, per-page meta overrides
- **Fast** — O(1) slug lookup, in-process templ rendering, gzip, ETag, content-hash cache busting
- **GDPR by design** — consent banner, first-party analytics proxy, no external CDN calls in default config
- **Access gate** — optional code-based protection for the full site or a specific path
- **Contact form** — built-in SMTP dispatch with spam filtering and rate limiting
- **Distroless container** — under 20 MB, no shell, no runtime overhead

## For Coding Agents

webhull ships a [Pi Skill](skills/webhull/SKILL.md) — load it and build production-ready sites without additional explanation.

```
skills/webhull/
├── SKILL.md                   # Quickstart, rules, enforcement gate
└── references/
    ├── CONFIG.md              # Full pages.yaml + config.yaml reference
    ├── PATTERNS.md            # Copy-paste configs for 4 site types
    └── WORKFLOW.md            # Step-by-step: empty dir → deployed site
```

See [AGENTS.md](AGENTS.md) for general coding-agent guidance.

## Quickstart

```bash
# Build
git clone https://github.com/layer87-labs/webhull.git
cd webhull
make build

# Run the example site
./build/bin/webhull -config example/site.yaml
# → open http://localhost:8080
```

## Configuration

All configuration lives in a single YAML file with `${VAR:default}` env-var expansion:

```yaml
site:
  name: "My Site"
  baseURL: "${BASE_URL:http://localhost:8080}"

i18n:
  defaultLanguage: "en"
  languages: ["en", "de"]

contentDir: "content"

server:
  port: "${PORT:8080}"
  environment: "${ENVIRONMENT:development}"
  staticDir: "static"
  staticCacheMaxAge: 8760h
```

Content files are HTML with YAML frontmatter:

```html
---
id: home
template: home
title: "Home"
seo_title: "My Site — Tagline"
startPage: true
heroLine1: "Hello,"
heroLine2: "world."
heroCTA1Text: "Get started"
heroCTA1Link: "/docs"
---

<!-- section[block]: What we do -->
<p>Body content here.</p>
```

## Documentation

Full documentation at [labs.layer87.de/docs/webhull](https://labs.layer87.de/docs/webhull/intro):

- [Quickstart](docs/webhull/getting-started/quickstart.md)
- [Configuration reference](docs/webhull/reference/configuration.md)
- [Content authoring](docs/webhull/content/authoring.md)
- [Templates](docs/webhull/content/templates.md)
- [SEO](docs/webhull/content/seo.md)
- [Access Gate](docs/webhull/features/access-gate.md)
- [Deployment](docs/webhull/reference/deployment.md)

## Container image

```dockerfile
FROM ghcr.io/layer87-labs/webhull:latest

COPY pages.yaml .
COPY content ./content
COPY static ./static
```

## License

Apache 2.0 — see [LICENSE](LICENSE).

Maintained by [Layer87](https://layer87.de).
