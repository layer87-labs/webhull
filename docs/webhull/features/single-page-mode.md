# Single-Page Mode

webhull supports a **single-page mode** where the entire site is one scrollable HTML
document permanently at `/`. There are no per-page slug routes. Language detection still
works — the same URL renders content in the visitor's language. Navigation items link to
`#anchor` sections within the page.

This is the standard pattern for landing pages, microsites, and single-brand studios.

---

## How to activate

Declare `slug: ""` in the frontmatter of the `home` page across **all** language
variants. webhull detects the empty slugs at startup and switches to single-page mode
automatically. No global flag is required. Multi-page sites are zero-change.

**Content file** (`content/de/index.html`):

```html
---
id: home
template: single
slug: ""
title: "Firma GmbH"
description: "Wir bauen großartige Dinge."
heroLine1: "Software,"
heroLine2: "die bleibt."
heroSubtitle: "Für Teams, die es ernst meinen."
heroCTA1Text: "Leistungen"
heroCTA1Link: "#leistungen"
heroCTA2Text: "Kontakt"
heroCTA2Link: "#contact"
---
<!-- section[services,id=leistungen]: Unsere Leistungen -->
<div class="service-card">
  <h3>Backend &amp; APIs</h3>
  <p>Skalierbare Go-Dienste, die unter Last stabil laufen.</p>
</div>

<!-- section[block,altbg,id=ueber-uns]: Über uns -->
<p>Gegründet 2021, mit Sitz in Berlin.</p>
```

The same `id: home` and `slug: ""` must appear in every language file
(`content/en/index.html`, etc.).

---

## `template: single`

The `single` template renders the complete one-page layout:

1. **Hero** — driven by frontmatter keys `heroLine1` / `heroLine2` (two-line gradient)
   or `heroTitle` (single-line). Optional `heroSubtitle`, `heroDescription`,
   `heroCTA1Text` / `heroCTA1Link`, `heroCTA2Text` / `heroCTA2Link`.
2. **Content sections** — all typed section markers in the body, rendered in declaration
   order. Each section gets an HTML `id` from `id=anchor` in the marker.
3. **Contact form** — appended automatically at `id="contact"` when
   `contact.enabled: true` in the site config.

### Typed section markers

```html
<!-- section[type,altbg,id=anchor]: Section Title -->
```

| Attribute | Values | Effect |
|---|---|---|
| `type` | `block`, `grid`, `services` | CSS layout of the inner wrapper |
| `altbg` | flag, no value | Adds `alt-bg` CSS modifier to the section |
| `id=anchor` | any string | Sets `id` on the `<section>` element for `#anchor` links |
| Title | raw HTML | Rendered in an `<h2>` section header; omit for no header |

---

## Navigation

Because the site has a single URL, navigation items use `#anchor` hrefs instead of
separate page slugs:

```yaml
navigation:
  header:
    de:
      - title: "Leistungen"
        url: "#leistungen"
      - title: "Über uns"
        url: "#ueber-uns"
```

The contact CTA in the header should point to `#contact`:

```yaml
ui:
  de:
    contactURL: "#contact"
    contactLabel: "Kontakt"
```

---

## Language switching

All language variants live at `/`. The language switcher sets a `lang` cookie by
navigating to `/?lang=<code>` — the i18n middleware reads the query param, persists the
cookie, and renders the correct language. No JavaScript required.

---

## SEO

| Aspect | Behaviour |
|---|---|
| Canonical URL | `/` (base URL) for all languages |
| `hreflang` | All alternates point to `/` — correct for cookie-negotiated content |
| Sitemap | One entry at `/` with the home page priority and changefreq |
| `og:type` | `website` (same as `home` and `legal` templates) |

---

## Contact form

When `contact.enabled: true`, the contact form is rendered automatically as the last
section of the page at `id="contact"`. No extra template work is needed.

The `contact.js` script is loaded automatically for `template: single` pages.

Link to it from the hero and nav with `#contact`.

---

## Example

A complete working example is in `example-single/`:

```
example-single/
  pages.yaml              # single-page site config
  content/
    de/index.html         # German single-page content (slug: "")
    en/index.html         # English single-page content (slug: "")
```

Run locally:

```bash
webhull -config deploy/config.yaml -pages example-single/pages.yaml
```
