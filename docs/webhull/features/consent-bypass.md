# Consent Bypass for Automated Audits

webhull includes a server-side mechanism that suppresses the GDPR cookie consent banner for automated tools (Lighthouse, Unlighthouse, Playwright, CI crawlers, etc.).

The banner is suppressed **at the server level** — no HTML is emitted and `consent.js` is not loaded. Real users are never affected.

---

## How it works

The `consent.Service` middleware inspects each incoming request for bypass signals **before** reading the consent cookie. When a signal is detected:

1. A `consent.State` with `Decided: true` and the appropriate categories is built in memory.
2. A `consent` cookie is written to the response (`SameSite=Lax`, no `Secure` requirement) so that follow-up requests from the same tool are handled without repeating the signal check.
3. The `Bypassed` flag is set on the state — the layout template uses this to omit the banner HTML and the `consent.js` script tag entirely.

---

## Bypass signals (priority order)

| Priority | Signal | Mode | Sent by |
|---|---|---|---|
| 1 | `Sec-Purpose: prefetch` header | `accept` | Chromium / Lighthouse (standard) |
| 2 | `X-Purpose: prefetch` header | `accept` | Legacy crawlers |
| 3 | `?consent=accept` query param | `accept` | Explicit opt-in (CI, Playwright) |
| 3 | `?consent=reject` query param | `reject` | Explicit opt-out (CI, Playwright) |

Header matching is **case-insensitive**. Query parameter matching is also case-insensitive (`?consent=ACCEPT` works).

**No bypass is triggered when the consent cookie is already decided** — returning users keep their existing consent state.

---

## Modes

### `accept` mode

All consent categories are set to `true`. Use this when your audit tool should see the full page as a consenting visitor would (analytics scripts active, etc.).

Cookie value written:
```json
{"decided": true, "categories": {"necessary": true, "analytics": true}}
```

### `reject` mode

Only `required` categories (e.g. `necessary`) are set to `true`; all others are `false`. Use this for privacy-first audits where you want to verify the site works without optional cookies.

Cookie value written:
```json
{"decided": true, "categories": {"necessary": true, "analytics": false}}
```

---

## Configuration for common tools

### Lighthouse (CLI)

Lighthouse sends `Sec-Purpose: prefetch` automatically for some sub-resources, but **not** for the initial page load. Use the `--extra-headers` flag:

```bash
lighthouse https://example.com \
  --extra-headers='{"Sec-Purpose":"prefetch"}' \
  --output html --output-path report.html
```

### Unlighthouse

In your `unlighthouse.config.ts`, pass the header via `puppeteerOptions`:

```ts
import { defineConfig } from 'unlighthouse'

export default defineConfig({
  site: 'https://example.com',
  puppeteerOptions: {
    extraHTTPHeaders: {
      'Sec-Purpose': 'prefetch',
    },
  },
})
```

Alternatively, use the query parameter approach (no Puppeteer config needed):

```ts
export default defineConfig({
  site: 'https://example.com/?consent=accept',
})
```

### Playwright

```ts
import { test } from '@playwright/test'

test.use({
  extraHTTPHeaders: {
    'Sec-Purpose': 'prefetch',
  },
})

// Or per-request:
test('audit page without consent banner', async ({ page }) => {
  await page.goto('https://example.com/?consent=accept')
  // Banner is gone — no need to click "Accept"
})
```

### curl / CI scripts

```bash
# Accept mode — all categories enabled
curl -H "Sec-Purpose: prefetch" https://example.com

# Reject mode — only necessary cookies
curl "https://example.com/?consent=reject"
```

---

## What is suppressed

| Element | Normal user (no cookie) | Bypass mode |
|---|---|---|
| `<div id="consent-banner">` | Rendered | **Not rendered** |
| `consent.js` script tag | Included | **Not included** |
| Consent cookie | Not set | Set in response |
| Analytics scripts | Blocked (no consent) | Depends on mode |

---

## Security notes

- The bypass **only** affects the consent banner — it does not bypass authentication, rate limiting, the gate, or any other security mechanism.
- The bypass cookie uses `SameSite=Lax` and no `Secure` flag so that local HTTP audit tooling works out of the box. For production deployments over HTTPS, this is safe because SameSite=Lax already prevents cross-site cookie sending.
- No bypass signal affects real users because real browsers do not send `Sec-Purpose: prefetch` on normal navigations.
