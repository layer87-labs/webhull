---
sidebar_position: 2
title: Cookie Consent
---

# Cookie Consent

webhull ships a GDPR cookie consent dialog. It is rendered server-side, gates the
analytics scripts, and can be reopened from any page so a decision can be
withdrawn.

For suppressing the dialog in automated audits, see
[Consent Bypass](./consent-bypass.md).

---

## Configuration

Consent lives in `pages.yaml`:

```yaml
consent:
  enabled: true
  categories:
    necessary:
      required: true   # cannot be switched off
      default: true
    analytics:
      required: false
      default: false   # opt-in, never pre-ticked
  i18n:
    de:
      title: "Cookie-Einstellungen"
      description: "Wir verwenden Cookies …"
      acceptAll: "Alle akzeptieren"
      rejectAll: "Alle ablehnen"
      customize: "Anpassen"
      save: "Auswahl speichern"
      categories:
        necessary: "Notwendig"
        analytics: "Analyse"
```

The `analytics` key is reserved (`consent.CategoryAnalytics`): it is what gates
the analytics scripts. Any other category is yours to define and act on.

Never set `default: true` on a non-required category — a pre-ticked box is not
consent.

---

## What the dialog does

| | |
|---|---|
| Rendered by | `internal/app/templates/consent.templ` |
| Behaviour | `internal/pkg/staticassets/js/consent.js` |
| Persisted as | `consent` cookie (`SameSite=Lax`, `Secure`, 1 year) + `POST /api/consent` |

The decision is written client-side first so it survives the reload that follows,
then mirrored to the server.

### Modal semantics

The dialog is a real modal, not a styled banner:

- focus moves into the dialog on open, so the title and description are announced first
- `Tab` and `Shift+Tab` are trapped inside it
- everything else in `<body>` is marked `inert` and `aria-hidden` while it is open, and restored exactly as it was on close
- background scrolling is locked
- focus returns to whatever was focused before it opened

### Escape

`Escape` behaves differently depending on why the dialog is open:

| Situation | `Escape` does |
|---|---|
| First visit, no decision yet | the same as **Reject all** |
| Reopened to review a decision | closes, decision unchanged |

Dismissing a first-visit dialog cannot leave the decision unmade — the dialog
would simply come back on the next page. Treating a dismissal as a refusal is the
privacy-preserving reading, and it mirrors the equally prominent reject button
rather than inventing a third outcome.

---

## Withdrawing consent

Consent that cannot be withdrawn as easily as it was given is not valid consent
(GDPR Art. 7(3)). webhull therefore keeps the dialog markup on every page after a
decision, hidden, ready to be reopened.

### Footer link

When consent is enabled, the footer renders a link that reopens the dialog. Its
label defaults to the consent banner title and can be overridden per language:

```yaml
ui:
  de:
    consentSettingsLabel: "Cookie-Einstellungen"
```

### Your own trigger

Any element carrying `data-consent-open` reopens the dialog — no JavaScript
needed on your side:

```html
<button type="button" data-consent-open>Cookie-Einstellungen</button>
```

### JavaScript API

```js
window.webhullConsent.open()    // open the dialog
window.webhullConsent.close()   // close without changing the decision
window.webhullConsent.state()   // { decided: bool, categories: { … } }
```

A reopened dialog starts with the category toggles expanded and pre-filled from
the stored decision, so it shows what is actually in effect. **Accept all** and
**Reject all** stay available as one-click shortcuts.

---

## Interaction with analytics

The analytics script tags are rendered server-side and only when `analytics`
consent has been accepted — see [Analytics](./analytics.md#consent-gating).

Because of that, a decision that flips the analytics category requires a page
reload to take effect. `consent.js` performs that reload automatically, in both
directions: granting consent reloads to bring the script in, withdrawing it
reloads to take the script out.

Without consent, webhull still records an anonymous server-side pageview, which
needs no cookie and no consent.

---

## Rendering rules

| Consent state | Banner markup | Visible | Analytics scripts |
|---|---|---|---|
| Disabled in config | — | — | rendered (nothing to gate) |
| Enabled, no decision | rendered | open | **not rendered** |
| Enabled, `analytics` rejected | rendered, `hidden` | reopenable | **not rendered** |
| Enabled, `analytics` accepted | rendered, `hidden` | reopenable | rendered |
| Bypassed (audit tools) | — | — | per bypass mode |

HTML responses carry `Vary: Cookie`, and the `ETag` differs per decision, so
intermediate caches cannot serve one visitor's consent state to another.
