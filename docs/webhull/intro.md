---
sidebar_position: 1
title: Introduction
---

# webhull

webhull is a Go binary that serves multilingual, SEO-optimised websites from YAML configuration and HTML content files. It produces a single statically-linked binary with no runtime dependencies — deploy it as a container, a systemd unit, or a plain executable.

## Why it exists

Most teams that need to host a website reach for a SaaS CMS, a cloud-vendor-managed platform, or a JavaScript framework with a server runtime. Each of those choices trades control for convenience: your content lives on someone else's infrastructure, your site depends on a runtime you don't fully understand, and GDPR compliance becomes an afterthought.

webhull was built on a different premise: **you should be able to serve a complete, SEO-optimised, multilingual website from a single binary and a YAML file, with no external dependencies at runtime.**

- No database. No admin UI. No user management.
- Content is YAML + HTML — version-controlled, diffable, reviewable.
- GDPR consent is built in, not bolted on.
- Analytics runs through your own proxy — no direct third-party calls from visitors.
- The entire system fits in a distroless container under 20 MB.

## Performance

webhull is designed to serve everything as fast as possible, at every layer.

### Request path

```
Request → Gin router (O(1) trie) → slug index (O(1) hash map)
        → template render (in-process, no subprocess, no network)
        → ETag check → 200 with gzipped HTML or 304 Not Modified (zero bytes)
```

- **O(1) page lookup** — all slugs are indexed into a hash map at startup. Every request resolves the page in constant time, regardless of how many pages exist.
- **In-process rendering** — [templ](https://github.com/a-h/templ) templates compile to Go functions. Rendering a page is a direct function call — no subprocess, no template engine, no network round-trip.
- **Sub-millisecond responses** — for a typical page with full layout, the render + gzip + write cycle completes in well under 1 ms under load.

### Gzip compression

All text responses (HTML, CSS, JS, sitemap, robots.txt) are gzip-compressed with `DefaultCompression`. Already-compressed formats (images, fonts, video, audio) are excluded automatically — no double-compression overhead.

### Caching strategy

webhull applies a tiered `Cache-Control` strategy without any configuration:

| Asset type | Cache-Control | Rationale |
|------------|--------------|-----------|
| CSS, JS | `public, max-age=31536000, immutable` | Content-hash in URL — safe to cache forever |
| Images (`/static/img/*`) | `public, max-age=2592000` | 30-day cache, no hash |
| HTML pages | `no-cache` | Always revalidate — ETag handles 304 |
| Sitemap, robots.txt | `public, max-age=86400` | 24-hour cache |

### ETag on every HTML response

Every rendered HTML page gets a `SHA-256 ETag` based on its content:

```
HTTP/1.1 200 OK
ETag: "a1b2c3d4e5f6a7b8"
Cache-Control: no-cache
```

On subsequent requests, the browser sends `If-None-Match: "a1b2c3d4e5f6a7b8"`. If the page hasn't changed, webhull returns `304 Not Modified` with **zero bytes transferred** — no re-render, no body, no bandwidth.

### Content-hash cache busting

Static asset URLs are automatically suffixed with a content hash at startup:

```html
<!-- in the HTML -->
<link rel="stylesheet" href="/static/css/style.css?v=a1b2c3d4">
<script src="/static/js/menu.js?v=e5f6a7b8"></script>
```

The hash is computed from the file content. When a file changes, the hash changes, the URL changes, and the browser fetches the new version — while unchanged files stay cached at the CDN or browser level indefinitely.

### Distroless container

The production image is built on [Chainguard's static distroless base](https://github.com/chainguard-images/images). The image contains only the compiled binary and your site content — no shell, no package manager, no OS tools.

- Container image: **under 20 MB**
- No runtime startup overhead (no JVM, no Node.js, no PHP-FPM)
- Attack surface is minimal: there is no shell to exploit

## Is this for you?

webhull is a good fit when:

- You need a **marketing site, documentation site, or client preview portal** where content changes are managed through Git, not a CMS dashboard.
- You want **full ownership of the stack** — no vendor lock-in, no cloud-managed hosting required.
- **Performance matters** — you want pages served in under a millisecond with full HTTP caching, gzip, and 304 support without any CDN configuration.
- **GDPR compliance** is a hard requirement and you want it handled at the framework level, not per-deployment.
- You need **multilingual support** (i18n) without a separate translation service.
- You want **built-in SEO** — sitemap, hreflang, structured data, open graph — without plugins.

webhull is not a good fit when:

- Non-technical editors need a visual drag-and-drop interface.
- Content is highly dynamic (personalised per user, real-time data).
- You need a full CMS with workflows, roles, and approvals.

## Architecture

```
site.yaml (config)        pages.yaml (structure)       content/de/*.html
      │                         │                              │
      └─────────────────────────┴──────────────────────────────┘
                                │
                         webhull binary
                                │
              ┌─────────────────┼─────────────────┐
              │                 │                 │
          /start           /leistungen        /kontakt
        (home.templ)    (default.templ)   (contact.templ)
              │
    sitemap.xml · robots.txt · /health · /metrics
              │
    Gzip · ETag · Cache-Control · Security headers
```

At startup, webhull:

1. Loads and validates configuration (operational + site structure).
2. Parses HTML content files, extracting YAML frontmatter and body sections.
3. Builds a slug index for O(1) page lookups.
4. Computes content hashes for all static assets (cache-busting).
5. Registers routes for every page slug in every configured language.
6. Serves pages rendered by [templ](https://github.com/a-h/templ) templates with gzip, ETag, and tiered Cache-Control.

## Links

- [GitHub](https://github.com/layer87-labs/webhull)
- [Quickstart](./getting-started/quickstart.md)
- [Configuration reference](./reference/configuration.md)
- [Access Gate](./features/access-gate.md)
