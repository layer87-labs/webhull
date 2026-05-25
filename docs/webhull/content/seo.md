---
sidebar_position: 3
title: SEO
---

# SEO

webhull treats SEO as a first-class concern. Every page gets a canonical URL, `hreflang` alternates, Open Graph tags, Twitter Card, and a `<meta name="author">` out of the box. The sitemap and `robots.txt` are auto-generated and always current.

## Meta tags (per-page)

Set SEO fields in content file frontmatter. All fields are optional — sensible defaults apply when omitted.

| Frontmatter key | Type | Default | Description |
|-----------------|------|---------|-------------|
| `seo_title` | string | `title` | Overrides `<title>` and `og:title`. Use for keyword-rich titles (30–60 chars). |
| `seo_description` | string | `description` | Overrides `<meta name="description">` and `og:description`. Target 70–160 chars. |
| `seo_priority` | float | `0.5` | Sitemap `<priority>`. Range 0.0–1.0. |
| `seo_changefreq` | string | `monthly` | Sitemap `<changefreq>`. Values: `always`, `hourly`, `daily`, `weekly`, `monthly`, `yearly`, `never`. |
| `seo_ogimage` | string | site default | Path (e.g. `/static/img/og/about.webp`) or absolute URL for `og:image`. |
| `seo_ogtype` | string | see below | OG type. Defaults to `website` for `home`/`legal` templates, `article` for all others. |
| `seo_noindex` | bool | `false` | Adds `<meta name="robots" content="noindex, nofollow">`. Use for gate pages, staging. |

**Example — fully annotated page:**

```html
---
id: about
template: default
title: "About"
description: "About Acme Corp — who we are and what we build."
seo_title: "About Acme Corp — Software built in Germany"
seo_description: "Acme Corp builds reliable backend systems operated entirely in Germany. Meet the team and learn about our philosophy."
seo_priority: 0.8
seo_changefreq: monthly
seo_ogimage: /static/img/og/about.webp
---
```

## Default OG image

Set a fallback `og:image` for all pages that do not specify `seo_ogimage`:

```yaml
seo:
  defaultOGImage: "/static/img/og/default.webp"
```

Relative paths are automatically prefixed with `site.baseURL`.

## Twitter Card

Default card type is `summary_large_image`. Override site-wide:

```yaml
seo:
  defaultTwitterCard: "summary"
```

## Author meta tag

`<meta name="author">` is automatically populated from `site.name`. No configuration required.

## Open Graph type defaults

| Template | Default `og:type` |
|----------|-------------------|
| `home` | `website` |
| `legal` | `website` |
| `default` | `article` |
| `contact` | `article` |

Override per page with `seo_ogtype`.

## Canonical URL

Every page gets `<link rel="canonical">` pointing to `{site.baseURL}/{slug}`. No configuration required.

## hreflang

For multilingual sites, webhull automatically generates `hreflang` alternate links for all language variants of each page. The page identified as the default language also gets `hreflang="x-default"`.

```html
<!-- generated automatically for a DE/EN site -->
<link rel="alternate" hreflang="de" href="https://example.com/ueber-uns"/>
<link rel="alternate" hreflang="en" href="https://example.com/about"/>
<link rel="alternate" hreflang="x-default" href="https://example.com/ueber-uns"/>
```

Pages are linked by their shared `id` field across language directories.

## Sitemap

`/sitemap.xml` is generated at startup from all non-`noindex` pages. It includes `<loc>`, `<lastmod>`, `<changefreq>`, `<priority>`, and `<xhtml:link>` alternates for multilingual pages.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:xhtml="http://www.w3.org/1999/xhtml">
  <url>
    <loc>https://example.com/home</loc>
    <lastmod>2026-05-25</lastmod>
    <changefreq>weekly</changefreq>
    <priority>1.0</priority>
    <xhtml:link rel="alternate" hreflang="de" href="https://example.com/start"/>
    <xhtml:link rel="alternate" hreflang="en" href="https://example.com/home"/>
  </url>
</urlset>
```

Pages with `seo_noindex: true` are excluded from the sitemap.

## robots.txt

`/robots.txt` is served automatically:

```
User-agent: *
Allow: /

Sitemap: https://example.com/sitemap.xml

Crawl-delay: 1
```

## JSON-LD / Structured Data

webhull supports JSON-LD structured data injection in two scopes.

### Global schemas (every page)

Define in `site.yaml` under `seo.globalJsonLD`. Use this for `Organization`, `WebSite`, or any schema that applies to all pages:

```yaml
seo:
  globalJsonLD:
    - |
      {
        "@context": "https://schema.org",
        "@type": "Organization",
        "name": "Acme Corp",
        "url": "https://example.com",
        "logo": "https://example.com/static/img/logo.webp"
      }
    - |
      {
        "@context": "https://schema.org",
        "@type": "WebSite",
        "name": "Acme Corp",
        "url": "https://example.com"
      }
```

### Per-page schemas

Define in `pages.yaml` under a page's `seo.jsonLD`. Use this for `SoftwareApplication`, `ContactPage`, `BreadcrumbList`, `FAQPage`, etc:

```yaml
pages:
  - id: product-alpha
    template: default
    seo:
      jsonLD:
        - |
          {
            "@context": "https://schema.org",
            "@type": "SoftwareApplication",
            "name": "Alpha",
            "applicationCategory": "BusinessApplication",
            "operatingSystem": "Cloud",
            "provider": {
              "@type": "Organization",
              "name": "Acme Corp",
              "url": "https://example.com"
            }
          }
    i18n:
      en:
        slug: product-alpha
        title: "Alpha"
        description: "Alpha by Acme Corp."
```

Global and per-page blocks are merged — global blocks appear first in the rendered HTML.

:::note
JSON-LD is not supported in HTML content file frontmatter. Use `pages.yaml` for per-page schemas.
:::

## Checklist before launch

- [ ] `seo_title` set on all pages (30–60 chars, keyword-rich)
- [ ] `seo_description` set on all pages (70–160 chars)
- [ ] `seo_priority` and `seo_changefreq` tuned per page importance
- [ ] `seo.defaultOGImage` configured (1200×630px minimum)
- [ ] Per-page `seo_ogimage` for high-traffic pages
- [ ] `Organization` JSON-LD in `seo.globalJsonLD`
- [ ] Page-type JSON-LD in `pages.yaml` for key pages
- [ ] `seo_noindex: true` on any staging / preview pages
