---
sidebar_position: 2
title: Templates
---

# Templates

webhull ships four built-in templates. The template is selected per page via the `template` frontmatter key. Every template receives the same `PageData` view model — the difference is which content keys each template reads and how they are laid out.

## `home`

The homepage template. Renders a full-width hero with animated background, CTAs, and flexible body sections below the fold.

**Frontmatter keys:**

| Key | Required | Description |
|-----|----------|-------------|
| `heroLine1` | yes | First line of the hero H1 heading. |
| `heroLine2` | yes | Second line — rendered with gradient styling. |
| `heroSubtitle` | no | Subtitle paragraph below the H1. |
| `heroDescription` | no | Longer description paragraph below the subtitle. |
| `heroCTA1Text` | no | Primary CTA button label. |
| `heroCTA1Link` | no | Primary CTA button href. |
| `heroCTA2Text` | no | Secondary CTA button label. |
| `heroCTA2Link` | no | Secondary CTA button href. |

**Body sections:**

If the body contains typed section markers (`<!-- section[type]: Title -->`), they are rendered in declaration order. Supported types: `block`, `grid`, `services`.

If no typed markers are present, the template falls back to legacy named section keys: `productsTitle`, `featuresTitle`, `focusTitle`, `targetGroupsTitle`, `differentiationTitle`, `surfaceTitle`, `servicesTitle` (each with a corresponding body key).

**Minimal example:**

```html
---
id: home
template: home
title: "Home"
description: "Welcome."
startPage: true
heroLine1: "Build fast."
heroLine2: "Ship confident."
heroSubtitle: "One binary. Zero dependencies."
heroCTA1Text: "Get started"
heroCTA1Link: "/docs"
---

<!-- section[block]: What we do -->
<p>We build software that runs reliably in production.</p>

<!-- section[grid,altbg]: Features -->
<div class="card"><h3>Fast</h3><p>Go binary, no Node runtime.</p></div>
<div class="card"><h3>Simple</h3><p>YAML + HTML, no CMS required.</p></div>
```

---

## `default`

Generic content page. Renders a small hero at the top and a content body below.

**Frontmatter keys:**

| Key | Required | Description |
|-----|----------|-------------|
| `heroTitle` | one of | Single-line H1 hero title. |
| `heroLine1` | one of | First line of a two-line H1 (use with `heroLine2`). |
| `heroLine2` | no | Second line — rendered with gradient styling. |
| `heroSubtitle` | no | Subtitle paragraph below the H1. |
| `breadcrumb` | no | Raw HTML for a breadcrumb nav shown above the H1. |

Use either `heroTitle` (single line) or `heroLine1`/`heroLine2` (two lines). If `heroTitle` is present it takes precedence.

**Body:**

The body content is rendered from `data.Content("body")`, or from typed/named section markers if present (same rules as `home`).

**Minimal example:**

```html
---
id: about
template: default
title: "About"
description: "About this project."
heroTitle: "About"
heroSubtitle: "What we are building and why."
---

<p>This is the about page body content.</p>
```

---

## `contact`

Contact page with hero, optional intro content, and a built-in contact form. The form submits to `/api/contact` via `POST` and handles validation, spam filtering, and SMTP dispatch.

**Frontmatter keys:**

Same hero keys as `default` — `heroTitle` or `heroLine1`/`heroLine2` + optional `heroSubtitle`.

**Body:**

Content rendered from `data.Content("body")` appears above the contact form. Use this for an intro paragraph, contact details, or a map embed.

**Form labels:**

Form labels default to built-in DE/EN strings based on the page language. Override any field via `ui.contactForm` in the site config:

```yaml
ui:
  de:
    contactForm:
      heading: "Schreib uns"
      submitText: "Absenden"
      successMsg: "Danke! Wir melden uns."
```

See [Contact Form](../features/contact-form.md) for full SMTP and form configuration.

**Minimal example:**

```html
---
id: contact
template: contact
title: "Contact"
description: "Get in touch."
heroLine1: "Let's talk."
heroSubtitle: "No pitch decks required."
---

<p>Send us a message and we will get back to you within one business day.</p>
```

---

## `legal`

Minimal template for imprint, privacy policy, and similar legal pages. Renders an H1 from `page.Title` followed by the body HTML. No hero, no sections — just clean prose.

**Frontmatter keys:**

No template-specific keys. `title` is rendered as the page H1.

**Body:**

The entire body is rendered as raw HTML. Standard HTML tags (`h2`, `h3`, `p`, `ul`, `table`) are all valid.

**Minimal example:**

```html
---
id: imprint
template: legal
title: "Imprint"
description: "Legal information."
seo_noindex: true
---

<h2>Company</h2>
<p>Acme Corp<br>123 Main Street<br>Anytown, AT 00000</p>

<h2>Contact</h2>
<p>hello@example.com</p>
```

---

## Adding a custom template

Custom templates are not currently supported as runtime plugins — the template set is compiled into the binary. To add a template:

1. Create `internal/app/templates/pages/mytemplate.templ`
2. Run `make generate` to compile it
3. Register the template name in `internal/app/server/handlers.go` `renderTemplate()`
4. Rebuild with `make build`

This keeps the binary hermetic and the template set auditable at compile time.
