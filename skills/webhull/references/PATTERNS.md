# Copy-Paste Patterns

Four real-world site configurations from production Webhull deployments.

---

## Pattern 1 — Multi-Page B2B Site

**Based on:** Layer87 Website (9 pages, corporate services, multi-nav)  
**Use when:** Professional B2B, multiple pages, contact form, analytics

### pages.yaml

```yaml
site:
  name: "Acme"
  baseURL: "https://acme.de"
  logoPath: "/static/img/logo.webp"
  faviconPath: "/static/img/favicon.webp"
  copyrightStartYear: 2025

i18n:
  defaultLanguage: "de"
  languages: ["de"]

contentDir: "content"

navigation:
  header:
    de:
      - title: "Services"
        url: "/services"
        slug: "services"
      - title: "Products"
        url: "/products"
        slug: "products"
      - title: "About"
        url: "/about"
        slug: "about"
  footer:
    de:
      - title: "Services"
        slug: "footer-services"
        url: "/services"
        children:
          - title: "Consulting"
            url: "/services#consulting"
          - title: "Development"
            url: "/services#development"
      - title: "Legal"
        slug: "footer-legal"
        url: "#"
        children:
          - title: "Imprint"
            url: "/imprint"
            slug: "imprint"
          - title: "Privacy"
            url: "/privacy"
            slug: "privacy"

contact:
  enabled: true
  recipientName: "Acme"
  recipients:
    - "contact@acme.de"
  subject:
    de: "[Contact Form] Message from {{.Name}}"
  maxLinks: 2
  rateLimit:
    requests: 3
    window: 15m

mail:
  from: "noreply@acme.de"
  fromName: "Acme"
  templates:
    de:
      subject: "Thank you for your message"
      bodyFile: "mail/de-confirmation.html"

consent:
  enabled: true
  categories:
    necessary:
      required: true
      default: true
    analytics:
      required: false
      default: false
  i18n:
    de:
      title: "Cookie Settings"
      description: "We use cookies for the proper operation of this website."
      acceptAll: "Accept all"
      rejectAll: "Reject all"
      customize: "Customize"
      save: "Save selection"
      categories:
        necessary: "Necessary"
        analytics: "Analytics"

analytics:
  plausible:
    enabled: true
    baseURL: "https://analytics.acme.de"
    domain: "acme.de"
    scriptPath: "/js/script.hash.outbound-links.js"

seo:
  defaultOGImage: "/static/img/og-default.png"
  defaultTwitterCard: "summary"
  globalJSONLD:
    - |
      {
        "@context": "https://schema.org",
        "@type": "Organization",
        "name": "Acme",
        "url": "https://acme.de",
        "email": "contact@acme.de"
      }

ui:
  de:
    contactURL: "/contact"
    contactLabel: "Contact"
    imprintURL: "/imprint"
    imprintLabel: "Imprint"
    privacyURL: "/privacy"
    privacyLabel: "Privacy Policy"
    footerTagline: "We build great software."
    allRights: "All rights reserved."
    contactForm:
      submitText: "Send message"
      fields:
        - name: name
          label: "Your name"
          type: text
          required: true
        - name: email
          label: "Email address"
          type: email
          required: true
        - name: company
          label: "Company"
          type: text
        - name: subject
          label: "Subject"
          type: text
          required: true
        - name: message
          label: "Message"
          type: textarea
          required: true
```

### Content structure

```
content/de/
├── home.html         (template: home, startPage: true)
├── services.html     (template: default)
├── products.html     (template: default)
├── about.html        (template: default)
├── contact.html      (template: contact)
├── imprint.html      (template: legal)
└── privacy.html      (template: legal)
```

---

## Pattern 2 — Single-Page Product Launch

**Based on:** ARCON (single-page landing, anchor nav, early-access form)  
**Use when:** Product launch, startup pitch, waitlist, minimal pages

### pages.yaml (key differences)

```yaml
site:
  name: "Product"
  baseURL: "https://product.acme.de"

navigation:
  header:
    de:
      - title: "Product"
        url: "/#product"
      - title: "How It Works"
        url: "/#how-it-works"
      - title: "Use Cases"
        url: "/#use-cases"
      - title: "Early Access"
        url: "/#access"
  footer:
    de:
      - title: "Product"
        slug: "footer-product"
        url: "#"
        children:
          - title: "Features"
            url: "/#product"
          - title: "How It Works"
            url: "/#how-it-works"
      - title: "Legal"
        slug: "footer-legal"
        url: "#"
        children:
          - title: "Imprint"
            url: "/imprint"
            slug: "imprint"
          - title: "Privacy"
            url: "/privacy"
            slug: "privacy"

contact:
  enabled: true
  recipients:
    - "hello@acme.de"
  subject:
    de: "[Early Access] Request from {{.Name}}"
```

### home.html pattern

```html
---
id: home
template: home
title: "Product"
heroLine1: "Product Name"
heroLine2: "by Acme"
heroSubtitle: "One-line value proposition."
heroCTA1Text: "Request Early Access"
heroCTA1Link: "/#access"
startPage: true
---

<section id="product">
  <h2>The Product</h2>
  <!-- feature cards -->
</section>

<section id="how-it-works">
  <h2>How It Works</h2>
  <!-- step-by-step -->
</section>

<section id="use-cases">
  <h2>Use Cases</h2>
  <!-- role-based use cases -->
</section>

<section id="access">
  <h2>Early Access</h2>
  <!-- CTA + contact form trigger -->
</section>
```

---

## Pattern 3 — Creative Portfolio

**Based on:** Studio OptiMayS (single-page, imagery-heavy, storytelling)  
**Use when:** Creative studio, photographer, designer portfolio

### pages.yaml (key differences)

```yaml
site:
  name: "Studio Name"
  baseURL: "https://studio.de"

navigation:
  header:
    de:
      - title: "About"
        url: "/#about"
      - title: "Work"
        url: "/#work"
      - title: "Contact"
        url: "/#contact"

contact:
  enabled: true
  recipients:
    - "hello@studio.de"
  subject:
    de: "[Contact] New message from {{.Name}}"

ui:
  de:
    contactForm:
      submitText: "Send →"
      fields:
        - name: name
          label: "Your name"
          type: text
          required: true
          placeholder: "Who are you?"
        - name: email
          label: "Your email"
          type: email
          required: true
          placeholder: "you@example.com"
        - name: message
          label: "Your project"
          type: textarea
          required: true
          placeholder: "Tell us about your vision..."
```

---

## Pattern 4 — Internal B2B Portal

**Based on:** Layer87 Marketplace (services catalog, dynamic pricing)  
**Use when:** Internal tool, service catalog, no public analytics/consent needed

### pages.yaml (key differences)

```yaml
site:
  name: "Portal"
  baseURL: "https://portal.acme.de"

navigation:
  header:
    de:
      - title: "Services"
        url: "/services"
        slug: "services"
      - title: "Pricing"
        url: "/pricing"
        slug: "pricing"
  footer:
    de:
      - title: "Acme"
        slug: "acme"
        url: "https://acme.de"
        children:
          - title: "acme.de"
            url: "https://acme.de"
          - title: "Contact"
            url: "mailto:contact@acme.de"

ui:
  de:
    contactURL: "mailto:contact@acme.de"
    contactLabel: "Contact"
    imprintURL: "https://acme.de/imprint"
    imprintLabel: "Imprint"
    privacyURL: "https://acme.de/privacy"
    privacyLabel: "Privacy Policy"
    footerTagline: "Internal services portal."

# All optional features disabled for internal use
consent:
  enabled: false

contact:
  enabled: false

analytics:
  plausible:
    enabled: false
```

---

## Confirmation Email Template

Minimal `mail/de-confirmation.html`:

```html
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: sans-serif; color: #333; max-width: 600px; margin: 0 auto; padding: 24px;">
  <h2>Thank you for your message</h2>
  <p>We have received your inquiry and will get back to you within 1–2 business days.</p>
  <p>Best regards,<br>The Team</p>
</body>
</html>
```

---

## config.yaml — Minimal Runtime Config

Works for all patterns above:

```yaml
server:
  port: "${PORT:8080}"
  environment: "${ENV:production}"

mail:
  host: "${SMTP_HOST}"
  port: ${SMTP_PORT:587}
  username: "${SMTP_USER}"
  password: "${SMTP_PASSWORD}"
  useTLS: ${SMTP_TLS:true}
```
