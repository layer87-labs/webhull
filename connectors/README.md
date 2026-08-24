# Connectors

Ready-to-use [plugin](../docs/webhull/features/plugins.md) manifests for common external
data sources — declarative HTTP connectors that fetch a JSON REST API in the background,
allowlist which fields leave the backend, and render an HTML fragment into a page. No
code, no compilation. Not tied to any one vertical — vehicle fleets, inventory feeds,
social media APIs, or any other JSON endpoint follow the same shape.

Contrast with `examples/plugins/example-http-source/`, which is a minimal didactic
example (public test API, no real-world quirks); the directories here are maintained,
real-world connectors with their own setup notes.

## Available connectors

| Connector | Source | Auth | Use case |
|---|---|---|---|
| [`rentandtravel/`](rentandtravel/) | rent and travel booking-engine API | none | Vehicle rental fleet listings |

## Using a connector

1. Copy the connector's directory into your site's `pluginsDir` (see `pages.yaml`), e.g.
   `site/plugins/rentandtravel/`.
2. Adjust `source` (station ID, locale, etc.) and `render.into` (your page's `id` and the
   `contentKey` your template reads) in `plugin.yaml` to your site.
3. Set any required env vars (see the connector's own README) — never edit secrets into
   the manifest directly; `plugin.yaml` loading rejects literal secret values.
4. Reference `data.Content("<contentKey>")` / `data.HasContent("<contentKey>")` from the
   target page, same as any other named content section.

## Adding a connector

Each connector is a self-contained directory: `plugin.yaml` + one render template + a
short `README.md` explaining what the source API is, what auth (if any) it needs, and any
quirks (rate limits, fields that don't render, etc.). PRs adding a new directory following
this shape are welcome.
