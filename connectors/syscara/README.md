# syscara connector

Fetches a dealer's public sales listings from the [Syscara](https://www.syscara.com/)
REST API and renders them as an HTML fragment (`sale.tmpl.html`): a card per vehicle
and a detail dialog with gallery, key facts and equipment.

## What it replaces

Syscara's own sales widget is a script tag (`widget.syscara.com/<id>.js`) that loads
about 170 KB of JavaScript, including its own jQuery, plus a stylesheet, and fetches the
listings from the visitor's browser. Its listing endpoint returns pre-rendered HTML, not
data, so it can't feed a plugin. This connector reads the REST API server-side instead:
the listing is native HTML in the site's own design, and the visitor's browser only
loads images from `cdn.syscara.com`.

Syscara's inquiry form is not rebuilt. The dialog's "Fahrzeug anfragen" button links to
the site's own contact form and passes the vehicle name as `?fahrzeug=<name>`.

## Setup

1. Request API credentials from Syscara support. The API is for the dealer's own use;
   check Syscara's terms if an agency operates the site on the dealer's behalf.
2. Copy this directory into your site's plugin directory, e.g. `site/plugins/syscara/`.
3. Copy `sale.js` into your site's static assets so it resolves at
   `/static/js/syscara-sale.js`. An inline `<script>` would be dropped by webhull's CSP.
4. Provide the credentials as environment variables. In Kubernetes, reference a Secret
   via `secretKeyRef`; never write them into `plugin.yaml` (the loader rejects literals).

   ```bash
   SYSCARA_USER=...
   SYSCARA_PASSWORD=...
   ```

5. Adjust `render.into.page` / `render.into.contentKey` in `plugin.yaml` to your page's
   frontmatter `id:` and content key, and the inquiry button's `href` in
   `sale.tmpl.html` to your contact form.
6. Style `.fleet-*` classes in your site's stylesheet; they are shared with the
   rentandtravel connector. The dialog adds `.fleet-dialog-equipment` (the
   equipment list) and `.fleet-dialog-price-note` (the VAT note after the price).

## Auth

HTTP Basic via `source.auth.basic` and `enrich.source.auth.basic`, both from
`SYSCARA_USER` / `SYSCARA_PASSWORD`. Credentials issued for a website may be limited to
certain endpoints; this connector needs `GET /sale/ads/` and `GET /data/media/`.

## Requests

One request to `/sale/ads/` plus one batch request per listing to `/data/media/` every
`refreshInterval` (15 minutes). Never on the request path.

## Data

| Shown | API field |
|---|---|
| Price | `prices.offer`, with `prices.vat` |
| Title | `model.producer`, `model.series`, `model.model` |
| First registration | `date.registration` |
| Mileage, power, gearbox, fuel | `mileage`, `engine.*` |
| Dimensions, gross weight | `dimensions.*`, `weights.total` |
| Beds, seats | `beds.*`, `seats` |
| Highlights | `features` (keys mapped to labels in `sale.js`) |
| Equipment | `texts.description` (HTML list, read as text) |
| Gallery | `media` → `/data/media/?file=path` → `images[].file` |

The API also returns purchase, list and B2B prices, discounts, accounting data, internal
identifiers and the seller. None of it is in `select.fields`, so none of it reaches the
page. Keep it that way when extending the allowlist.

## Known quirks

- `/sale/ads/` returns an object keyed by listing id, not an array, which is why the
  manifest relies on webhull's keyed-collection support.
- Images are referenced by media id only. `file=path` resolves them to CDN URLs ending
  in `/1500.jpg`; the same path also serves `sm`, `250` and `500`.
- `/data/media/` answers in the order of the requested ids, so the gallery follows the
  dealer's image order. Floor plans (`group: layout`) are moved to the end.
