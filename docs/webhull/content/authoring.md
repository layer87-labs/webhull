---
sidebar_position: 1
title: Content Authoring
---

# Content Authoring

Pages are HTML files with a YAML frontmatter block. The binary scans a content directory at startup, parses every file, and registers a route for each page slug.

## Directory structure

```
content/
  de/
    home.html       → slug: "home",   lang: de
    kontakt.html    → slug: "kontakt", lang: de
  en/
    home.html       → slug: "home",   lang: en
    contact.html    → slug: "contact", lang: en
```

- The **language code** is the directory name (`de`, `en`, `fr`, …).
- The **slug** is the filename without `.html` — unless the frontmatter sets `slug` explicitly (see [nested paths](#nested-paths)).
- For multilingual sites, each language directory must contain a file for every page.
- Language directories are scanned **flat**. Subdirectories inside `content/de/` are ignored (with a startup warning); they are neither pages nor languages. Nested URLs come from the `slug` key, not from the directory tree.

## Frontmatter

Every file starts with a YAML frontmatter block delimited by `---`:

```html
---
id: contact
template: contact
title: "Contact"
description: "Get in touch with us."
---

<!-- body HTML here -->
```

### Reserved frontmatter keys

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `id` | string | recommended | Internal page identifier, shared across languages. Defaults to the slug if omitted. |
| `slug` | string | no | URL path override. Defaults to the filename without `.html`. May contain `/` for [nested paths](#nested-paths) (`produkte/desk` → `/produkte/desk`). `""` puts the page at `/` ([single-page mode](../features/single-page-mode.md), `id: home` only). |
| `template` | string | no | Template to render. One of `home`, `default`, `contact`, `legal`. Defaults to `default`. |
| `title` | string | yes | Page title used in `<title>` tag (unless `seo_title` overrides it). |
| `description` | string | yes | Meta description (unless `seo_description` overrides it). |
| `keywords` | string | no | Meta keywords (comma-separated). |
| `startPage` | bool | no | Mark this page as the start page for its language. Root `/` redirects here. Set on exactly one page per language. |
| `seo_title` | string | no | Overrides `title` in `<title>`, `og:title`. See [SEO](./seo.md). |
| `seo_description` | string | no | Overrides `description` in meta description and `og:description`. |
| `seo_priority` | float | no | Sitemap priority (0.0–1.0). Default: `0.5`. |
| `seo_changefreq` | string | no | Sitemap change frequency. Default: `monthly`. |
| `seo_ogimage` | string | no | Absolute path or URL for `og:image`. |
| `seo_ogtype` | string | no | OG type. Defaults to `website` for `home`/`legal`, `article` for others. |
| `seo_noindex` | bool | no | Add `<meta name="robots" content="noindex, nofollow">`. |

All other keys are passed through to the template as content values accessible via `data.Content("key")`.

### Content keys (template-specific)

Keys not in the reserved list above become available in templates as `data.Content("key")`. See [Templates](./templates.md) for which keys each template uses.

```html
---
id: home
template: home
title: "Home"
description: "Welcome to my site."
startPage: true
heroLine1: "Simple."
heroLine2: "Fast."
heroSubtitle: "Built with webhull."
heroCTA1Text: "Get started"
heroCTA1Link: "/docs"
---
```

## Body formats

The HTML after the closing `---` delimiter is the page body. Three formats are supported — pick one per file.

### Plain body

The entire body is accessible as `data.Content("body")` in the template:

```html
---
id: about
template: default
title: "About"
description: "About this project."
heroTitle: "About"
---

<p>This is the about page.</p>
<p>Any HTML is valid here.</p>
```

### Named sections (legacy)

Split the body into named sections using `<!-- section: name -->` markers:

```html
---
id: about
template: default
title: "About"
description: "About this project."
---

<!-- section: body -->
<p>Main content here.</p>

<!-- section: sidebar -->
<p>Sidebar content here.</p>
```

Each section is accessible in the template as `data.Content("body")`, `data.Content("sidebar")`, etc.

### Typed sections (recommended)

For structured layouts — use `<!-- section[type,options]: Title -->` markers. Sections are rendered in declaration order by the template.

```html
---
id: services
template: default
title: "Services"
description: "What we offer."
heroTitle: "Services"
---

<!-- section[block]: Our Approach -->
<p>We focus on long-term partnerships.</p>

<!-- section[grid,altbg]: What We Offer -->
<div class="card">
  <h3>Consulting</h3>
  <p>Architecture and strategy.</p>
</div>
<div class="card">
  <h3>Development</h3>
  <p>From prototype to production.</p>
</div>

<!-- section[services,altbg,id=tech]: Technology -->
<div class="service-card">...</div>
```

**Marker syntax:**

```
<!-- section[type]: Title HTML -->
<!-- section[type,altbg]: Title HTML -->
<!-- section[type,altbg,id=anchor]: Title HTML -->
```

| Part | Values | Description |
|------|--------|-------------|
| `type` | `block`, `grid`, `services` | Controls CSS layout class applied to the section |
| `altbg` | flag (no value) | Applies `alt-bg` CSS modifier for alternating background |
| `id=anchor` | string | Sets the HTML `id` on the section element (for anchor links) |
| Title | HTML string | Rendered as `<h2>` inside the section header. Empty = no header. |

Typed sections are available in `data.Page.Sections` as an ordered slice.

## Nested paths

A page can live beneath another page in the URL — `/produkte/desk` under `/produkte` — without any directory nesting on disk. Keep the file flat in the language directory and set the `slug` key:

```
content/
  de/
    produkte.html         → slug: "produkte"        → /produkte
    produkte-desk.html    → slug: "produkte/desk"   → /produkte/desk
  en/
    products.html         → slug: "products"        → /products
    products-desk.html    → slug: "products/desk"   → /products/desk
```

```html
---
id: product-desk
template: default
title: "Desk"
description: "Groupware for small teams."
slug: produkte/desk
---
```

The filename is then only a name for humans; pick one that mirrors the URL (`produkte-desk.html`) so the tree stays readable. The parent page (`/produkte`) does not have to exist — a nested slug is just a slug with a slash in it.

Everything downstream is derived from the slug and works unchanged: the route (`/produkte/desk`, with `/produkte/desk/` redirecting to it), the canonical URL, `hreflang` alternates via the shared `id`, the language switcher, and the sitemap entry.

**Slug rules** (checked at startup, per language file):

| Rule | Example | Result |
|------|---------|--------|
| slashes between segments | `produkte/desk` | valid |
| no leading slash | `/produkte/desk` | error |
| no trailing slash | `produkte/desk/` | error |
| no empty segment | `produkte//desk` | error |
| no `.` or `..` segment | `produkte/../desk` | error |
| no whitespace | `produkte desk` | error |
| no `?` or `#` | `produkte#top` | error |

Slugs must be unique across **all** languages, not only within one: the URL carries no language prefix, so `desk` in `de/` and `desk` in `en/` would fight over the same route. This is also a startup error.

**Navigation active state:** a header or footer item is active on the page it points to *and* on every page nested beneath it, matched on path-segment boundaries. With `slug: produkte` in the navigation, `/produkte/desk` highlights "Produkte"; `/produkte-alt` does not. See [Configuration → navigation](../reference/configuration.md#navigation--header-and-footer).

## Multilingual pages

Pages are matched across languages by their `id` field. If all language files share the same `id`, webhull automatically generates `hreflang` alternate links in the `<head>`.

```
content/de/contact.html  →  id: contact  →  slug: kontakt
content/en/contact.html  →  id: contact  →  slug: contact
```

The `id` ties the two files together. The slugs can differ per language.

If a language file is missing for a configured language, startup fails with:

```
page "contact" missing i18n for language "en"
```

## Validation at startup

webhull validates all content at startup:

- Every page must have an entry for every configured language.
- Every slug must be well-formed (see [nested paths](#nested-paths)).
- No two pages may share the same slug — across all languages, since the URL has no language prefix.
- Only the page with `id: home` may use an empty slug (single-page mode).
- A page with `id: home` must exist — it is the start page every language redirects to from `/`.
