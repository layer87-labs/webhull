---
sidebar_position: 7
title: Plugins
---

# Plugins

Declarative data-source plugins: fetch JSON from an external REST API on a background
schedule, allowlist which fields leave the backend, render an HTML fragment, and inject
it into a page's content — no code, no compilation, no container rebuild to swap the
data source. Not tied to any one API or vertical: a rental fleet, an inventory feed, a
social media API, or any other JSON endpoint can be wired up the same way.

## How it works

```
plugins/
  rentandtravel/
    plugin.yaml            ← manifest: source, field allowlist, render target
    fleet.tmpl.html         ← html/template fragment
```

At startup, webhull discovers every `plugin.yaml` under `pluginsDir`, validates it, does
one bounded initial fetch, and starts a background refresh loop. Requests never trigger a
fetch — `buildPageData()` only reads the last successfully rendered fragment from memory.
A page whose frontmatter references the plugin's `contentKey` renders through the
existing `Content()` / `HasContent()` methods, same as any other content section — **no
template changes are required** to consume a plugin.

```
background: fetch → allowlist fields → render html/template → cache
request:    Page.ID → cached HTML → Content(key) → @templ.Raw(...)
```

## Manifest

```yaml
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: rentandtravel

source:
  url: https://api.example.com/v1/articles
  query:
    locale: de-DE
    station: "5619"
  headers:
    Authorization: "${RNT_TOKEN}"     # optional — must be exactly ${VAR} or ${VAR:default}
  timeout: 8s                          # default 8s
  refreshInterval: 15m                 # default 15m, minimum 30s
  staleWhileError: 24h                 # default 24h

select:
  root: articles                       # dot path to the array in the response; empty = response is the array
  fields:                              # allowlist — nothing else reaches the template
    - id
    - make
    - model
    - images.outside.medium

render:
  template: fleet.tmpl.html            # resolved relative to plugin.yaml
  into:
    page: vermietung                   # the page's stable id (frontmatter "id:"), not its slug
    contentKey: fleet                  # referenced via data.Content("fleet") in the page template

csp:
  imgSrc:
    - https://cdn.example.com          # exact origins only — extends img-src
```

### Security rules enforced at load time

- **`select.fields` cannot be empty.** Deny by default — a plugin with no allowlist has
  nothing to render, on purpose.
- **`source.headers` values must be exactly `${VAR}` or `${VAR:default}`.** A literal
  value (or a value with anything else mixed in) is a startup error — the manifest is the
  one place a secret could accidentally get committed, so partial matches are rejected,
  not just obvious ones.
- **`csp.imgSrc` entries must be exact origins, no wildcards.**
- **Two plugins cannot target the same `page`/`contentKey`**, and plugin `name`s must be
  unique — both are startup errors.
- **`render.template` must exist next to `plugin.yaml`** at load time.

### Field allowlisting

`select.fields` lists dot paths into each item's JSON object, e.g.
`images.outside.medium`. Only listed paths are extracted; everything else in the
upstream response — internal IDs, session tokens, unrelated metadata — never leaves the
allowlist step. A path that doesn't resolve for a given item is simply omitted, not an
error (upstream APIs don't always send every field for every item).

## Rendering

`render.template` is an [`html/template`](https://pkg.go.dev/html/template) file,
auto-escaping every interpolated value — allowlisted-but-untrusted upstream data (titles,
labels, image URLs) can never break out of the surrounding markup. It receives:

```go
type renderData struct {
    Items []Item // Item is map[string]interface{}, keyed by the exact dot path from select.fields
}
```

```html
{{range .Items}}
<article class="fleet-card">
  <img src="{{index . "images.outside.medium"}}" alt="{{index . "make"}} {{index . "model"}}">
  <h3>{{index . "make"}} {{index . "model"}}</h3>
</article>
{{end}}
```

Small helpers are available: `hasSuffix`, `contains`, `default`.

## Configuration

```yaml
# pages.yaml — plugins are part of a site's content, like contentDir
pluginsDir: "plugins"
```

Resolves relative to the file that defines it, same rule as `contentDir`. A missing
`pluginsDir` (or the directory not existing) is not an error — plugins are opt-in.

## Consuming a plugin from a page

There are two ways to pull a plugin's rendered fragment into a page, depending on the
page's content format (see [Content Templates](../content/templates.md)):

**A page using typed or named sections** — call `data.Content("fleet")` /
`data.HasContent("fleet")` from the `.templ` template, exactly like any other named
content key. This is the only option available to a `.templ` file, since sections are
resolved before the template runs.

**A raw-body page** (a content file with no section markers — the whole file is one
custom HTML document, resolved via `Content("body")`) — write an inline marker at the
injection point:

```html
---
id: vermietung
template: single
slug: "vermietung"
title: "Vermietung"
---
<section>
  <h2>Fahrzeuge &amp; Konditionen</h2>
  <!-- plugin: fleet -->
</section>
```

`Content(key)` scans whatever string it returns — including the entire raw `"body"`
value — for `<!-- plugin: NAME -->` markers and replaces each with that plugin's
rendered fragment (empty string if the plugin has no data yet, e.g. before its first
fetch completes). This mirrors the existing `<!-- section: name -->` marker syntax and
needs no page-template changes: it works for any content key, on any template, including
fully custom pages that manage their own HTML structure end to end.

## Operational notes

- **Server-side only.** The visitor's browser never talks to the upstream API directly —
  no consent gating needed for the data fetch itself, no CORS concerns, no layout shift.
  Only images referenced by the rendered fragment are loaded browser-side, from the host
  listed in `csp.imgSrc`.
- **Stale-while-error.** If fetches start failing, the last good fragment keeps being
  served for `staleWhileError` (default 24h) before the content key goes empty — a
  temporary upstream outage doesn't take down the page section.
- **Graceful shutdown.** Every plugin's background refresh loop is stopped as part of the
  server's normal shutdown sequence, alongside `Analytics.Close()`.
