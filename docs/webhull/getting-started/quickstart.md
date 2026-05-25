---
sidebar_position: 1
title: Quickstart
---

# Quickstart

Build and run a working webhull site in five minutes.

## Prerequisites

- Go 1.22+
- [templ](https://templ.guide) CLI: `go install github.com/a-h/templ/cmd/templ@latest`
- `make` (standard on Linux/macOS)

## 1. Clone and build

```bash
git clone https://github.com/layer87-labs/webhull.git
cd webhull
make build
```

The binary is written to `build/bin/webhull`.

## 2. Create a minimal site config

Create `mysite/site.yaml`:

```yaml
site:
  name: "My Site"
  baseURL: "http://localhost:8080"
  logoPath: "/static/img/logo.webp"
  faviconPath: "/static/img/favicon.webp"

i18n:
  defaultLanguage: "en"
  languages: ["en"]

contentDir: "content"

server:
  port: "8080"
  environment: "development"
  staticDir: "static"
```

## 3. Create a home page

Create `mysite/content/en/home.html`:

```html
---
id: home
template: home
title: "Home"
description: "My first webhull site."
startPage: true
heroLine1: "Hello,"
heroLine2: "world."
heroSubtitle: "A webhull site running locally."
heroCTA1Text: "Learn more"
heroCTA1Link: "/about"
---
```

## 4. Run

```bash
./build/bin/webhull -config mysite/site.yaml
```

Open [http://localhost:8080](http://localhost:8080) — you will be redirected to `/home` and see the home page rendered with the default layout.

## 5. Add a second page

Create `mysite/content/en/about.html`:

```html
---
id: about
template: default
title: "About"
description: "What this site is about."
heroTitle: "About"
heroSubtitle: "A simple example page."
---

<p>This is the about page body. Add any HTML here.</p>
```

Restart the server — `/about` is now live.

## Development mode (hot reload)

For active development, use [air](https://github.com/air-verse/air) for hot reload:

```bash
make dev
```

This watches `.go`, `.templ`, `.yaml`, and `.html` files and rebuilds on change.

## Next steps

- [Configuration model](./configuration-model.md) — understand monolithic vs split config and env-var expansion.
- [Content authoring](../content/authoring.md) — all frontmatter keys, section markers, and i18n.
- [Templates](../content/templates.md) — which content keys each template expects.
- [Deployment](../reference/deployment.md) — Helm, container image, production checklist.
