# rentandtravel connector

Fetches a station's rentable vehicle fleet from the
[rent and travel](https://www.rentandtravel.de/) booking-engine API and renders it as an
HTML fragment (`fleet.tmpl.html`).

## What it replaces

Sites using rent and travel typically embed a booking widget as an iframe
(`cdn.be.rentandtravel.de/wl/Plugin/App/app.js` loading `wl.be.rentandtravel.de`). This
connector reads the same underlying data server-side and renders it as native HTML the
site fully controls — same design system, no layout shift, no third-party request from
the visitor's browser for the listing itself (only vehicle images load from
`cdn.be.rentandtravel.de`, declared via `csp.imgSrc`).

The vendor's own booking/payment flow (Stripe checkout) is untouched. Each card opens a
detail dialog — a photo gallery and full specs, built entirely from data already fetched
with the list (no per-vehicle API call) — whose "Jetzt buchen" button links out to that
vehicle's real booking page on `wl.be.rentandtravel.de`, built from the item's own `id`
and `station.id`. No separate detail route on the consuming site.

## Setup

1. Find your **station ID**. It's the `data-stationid` attribute on the vendor's iframe
   loader snippet (`<div id="app-booking-container" data-stationid="1234">`), or ask
   rent and travel support.
2. Copy this directory into your site's plugin directory, e.g.
   `site/plugins/rentandtravel/`.
3. Set the env var (or edit the default inline in `plugin.yaml`):

   ```bash
   RNT_STATION_ID=1234
   ```

4. Adjust `render.into.page` / `render.into.contentKey` in `plugin.yaml` to match your
   page's frontmatter `id:` and the content key your template reads.
5. Style `.fleet-*` and `.fleet-dialog*` classes in your site's stylesheet — the template
   ships unstyled, class-only markup. The dialog uses the native HTML `<dialog>` element
   (`showModal()`); on a browser without dialog support the "Details ansehen" button
   simply does nothing — the card's own summary data is still fully visible, so nothing
   is lost, just the gallery.

## Auth

**None.** This endpoint is the same public data the vendor's own iframe reads —
verified with `curl`, no `Authorization` header or cookie required, and the response
carries a permissive CORS header. If rent and travel changes this in the future, add a
`source.headers.Authorization: "${RNT_TOKEN}"` line to `plugin.yaml` and set the env var
— never a literal token (the loader rejects that at startup).

## Configuration reference

| Env var | Default | Meaning |
|---|---|---|
| `RNT_STATION_ID` | `5619` | Rental station ID |
| `RNT_LOCALE` | `de-DE` | Response locale |
| `RNT_VEHICLE_TYPE` | `motorhome` | Vehicle category filter |
| `RNT_PAGE_SIZE` | `24` | Max vehicles fetched |

## Known quirks

- The upstream feed serves `.tif` images for some categories (`interieur`, `bath`,
  `bed`). The card image (`images.outside.medium`) is consistently `.png`/`.jpg`; the
  detail dialog's gallery uses the full `images.all` array and filters `.tif` entries
  out client-side (they're already in the page's data, filtering them server-side would
  only save a few hundred bytes at the cost of a second, non-trivial field-path shape).
- `pricePerNightFrom` has no currency/decimal formatting from the API — the template
  assumes EUR, whole numbers (`110` → "ab 110 €").
- Not every vehicle sends every field (`intro1`, `labels` are often empty) — the
  template guards each with `{{if}}`/`{{with}}` rather than assuming presence.
