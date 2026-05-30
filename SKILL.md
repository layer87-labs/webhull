# Webhull — Skill für Coding Agents

Diesen Skill verwenden, um mit **Webhull** vollständige, produktionsreife Webseiten zu bauen — ohne Webhull selbst zu erweitern.

---

## 🎯 WANN WEBHULL VERWENDEN

Webhull ist das richtige Framework, wenn der Agent...

- **Eine statische Website bauen** soll (Corporate Site, Landing Page, Portfolio, Dokumentation)
- **Server-Side Rendering** mit SEO-Optimierung braucht (Sitemap, robots.txt, hreflang, JSON-LD, Meta-Tags)
- **Multi-Language (i18n) Support** benötigt ohne JS-Framework
- **Contact Forms mit SMTP-Validierung** einbauen soll
- **Cookie Consent & GDPR-Compliance** benötigt
- **Privacy-friendly Analytics** (Plausible) verwenden möchte
- **Minimal Dependencies** will — nur Go Binary, keine Node/npm/webpack nötig
- **Container-Ready Deployment** bevorzugt (Helm + Kubernetes)
- **Single Binary** für einfaches Deployment will

### Webhull ist NICHT geeignet für:

❌ SPAs mit Client-Side Routing  
❌ Real-time Applications (WebSockets)  
❌ Database-driven Content (Webhull ist stateless)  
❌ User Authentication (nur einfacher HMAC-basierter Gate möglich)  
❌ Complex Interactive Features (nur Vanilla JS)  

---

## ⚡ SCHNELLSTART (5 Minuten)

### 1. Projektstruktur erstellen

```bash
# Neues Projekt-Verzeichnis
mkdir my-website && cd my-website

# Webhull Site-Struktur vorbereiten
mkdir -p site/{content/de,mail,static/{css,fonts,js,img}}

# Containerfile + Docker Compose
cat > Containerfile << 'EOF'
FROM ghcr.io/layer87-labs/webhull:1.2.1
COPY site /app/site
EOF

cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  webhull:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./site:/app/site
    environment:
      SMTP_HOST: "mail.example.de"
      SMTP_USER: "user@example.de"
      SMTP_PASSWORD: "secret"
      PORT: "8080"
  
  mailpit:  # Local test mailserver
    image: axllent/mailpit:latest
    ports:
      - "1025:1025"
      - "8025:8025"
EOF
```

### 2. Minimale `site/pages.yaml`

```yaml
site:
  name: "My Company"
  baseURL: "https://my-company.de"
  copyrightStartYear: 2025

i18n:
  defaultLanguage: "de"
  languages: ["de"]

contentDir: "content"

navigation:
  header:
    de:
      - title: "Home"
        url: "/"
        slug: "home"
      - title: "Contact"
        url: "/contact"
        slug: "contact"
  footer:
    de: []

ui:
  de:
    contactURL: "/contact"
    contactLabel: "Kontakt"
    footerTagline: "Welcome"
    imprintURL: "/imprint"
    privacyURL: "/privacy"

contact:
  enabled: true
  recipientName: "Support"
  recipients:
    - "support@my-company.de"
  subject:
    de: "[Contact] Nachricht von {{.Name}}"
```

### 3. Minimale `site/config.yaml` (Runtime)

```yaml
server:
  port: "${PORT:8080}"

mail:
  host: "${SMTP_HOST:localhost}"
  port: ${SMTP_PORT:1025}
  username: "${SMTP_USER}"
  password: "${SMTP_PASSWORD}"
  useTLS: ${SMTP_TLS:true}
```

### 4. Content-Dateien (HTML-Fragmente)

**`site/content/de/home.html`**

```html
---
id: home
template: home
title: "Startseite"
seo_title: "My Company - Home"
seo_description: "Welcome to my company"
heroLine1: "Welcome to"
heroLine2: "My Company"
heroSubtitle: "We build great things."
heroCTA1Text: "Learn More"
heroCTA1Link: "/about"
heroCTA2Text: "Contact Us"
heroCTA2Link: "/contact"
---

<section id="hero">
  <h1>Welcome</h1>
  <p>We create great products and services.</p>
</section>

<section id="about">
  <h2>About Us</h2>
  <p>Founded in 2025...</p>
</section>
```

**`site/content/de/contact.html`**

```html
---
id: contact
template: contact
title: "Kontakt"
seo_title: "Contact us"
seo_description: "Get in touch with our team"
---

<section id="contact">
  <h1>Get in Touch</h1>
  <p>We'd love to hear from you.</p>
  <!-- Contact form wird auto-rendered -->
</section>
```

**`site/mail/de-confirmation.html`**

```html
<p>Thank you for your message. We will get back to you soon.</p>
```

### 5. Starten & Testen

```bash
# Build & Run (local development)
docker-compose up --build

# Open browser: http://localhost:8080
# Mailpit UI: http://localhost:8025
```

✅ **Die Website läuft!**

---

## 📖 KONFIGURATIONSREFERENZ

### `site/pages.yaml` — Build-Time Config (im Container gebacken)

#### `site` — Site-Identität

```yaml
site:
  name: "Company Name"                    # Required | Wird in Meta-Tags, Copyright verwendet
  baseURL: "https://company.de"           # Required | Für Sitemap, Canonical Tags, OG-Tags
  logoPath: "/static/img/logo.webp"       # Optional | Header Logo
  faviconPath: "/static/img/favicon.ico"  # Optional | Browser Tab Icon
  copyrightStartYear: 2025                # Optional | © 2025-2026 in Footer
```

#### `i18n` — Internationalisierung (Multi-Language)

```yaml
i18n:
  defaultLanguage: "de"                   # Required | Fallback-Sprache
  languages: ["de", "en", "fr"]           # Required | Verfügbare Sprachen
  # Webhull erstellt auto: /de/, /en/, /fr/ Subpfade
  # Hreflang-Tags für SEO werden auto-generiert
```

**Wichtig:** Für jede Sprache in `languages` müssen `content/[lang]/`-Verzeichnisse existieren.

#### `contentDir` — Content-Verzeichnis

```yaml
contentDir: "content"                     # Optional | Default: "content"
# Webhull sucht: content/[language]/[slug].html
```

#### `navigation` — Header + Footer Navigation

```yaml
navigation:
  header:
    de:                                   # Language key
      - title: "Home"
        url: "/"
        slug: "home"
        children:                         # Optional nested items
          - title: "Subpage"
            url: "/subpage"
            slug: "subpage"
    en:                                   # English nav
      - title: "Home"
        url: "/"
        slug: "home"
  
  footer:
    de:
      - title: "Company"
        slug: "footer-company"
        url: "#"
        children:
          - title: "About"
            url: "/about"
            slug: "about"
          - title: "Team"
            url: "/team"
            slug: "team"
```

**Active State:** Webhull markiert automatisch das aktuelle Menü-Item.

#### `contact` — Contact Form Konfiguration

```yaml
contact:
  enabled: true                           # true/false | Contact Form aktivieren
  recipientName: "Support Team"           # Wird in Bestätigungs-Mail verwendet
  recipients:                             # Required | Email-Adressen für Formular-Einträge
    - "support@company.de"
    - "contact@company.de"
  subject:                                # Required | Subject-Zeile per Sprache
    de: "[Contact Form] Neue Nachricht von {{.Name}}"
    en: "[Contact Form] New message from {{.Name}}"
  maxLinks: 2                             # Optional | Max Links in Message (Spam-Filter)
  rateLimit:
    requests: 3                           # Optional | Max Requests pro...
    window: 15m                           # Optional | ...Zeitfenster (Spam-Filter)
```

#### `mail` — Email-Konfiguration (SMTP + Templates)

```yaml
mail:
  from: "noreply@company.de"              # Required | Sender-Adresse
  fromName: "Company"                     # Optional | Sender-Name
  templates:
    de:
      subject: "Vielen Dank für deine Nachricht"
      bodyFile: "mail/de-confirmation.html"  # Path rel. to site/ dir
    en:
      subject: "Thank you for your message"
      bodyFile: "mail/en-confirmation.html"
```

**Note:** SMTP-Credentials kommen aus `config.yaml` (runtime).

#### `consent` — GDPR Cookie Consent Banner

```yaml
consent:
  enabled: true                           # true/false
  categories:
    necessary:                            # IMMER erforderlich
      required: true
      default: true
    analytics:                            # Optional Analytics (z.B. Plausible)
      required: false
      default: false
    marketing:                            # Optional Marketing/Tracking
      required: false
      default: false
  i18n:
    de:
      title: "Cookie-Einstellungen"
      description: "Wir verwenden Cookies..."
      acceptAll: "Alle akzeptieren"
      rejectAll: "Alle ablehnen"
      customize: "Anpassen"
      save: "Auswahl speichern"
      categories:
        necessary: "Notwendig"
        analytics: "Analyse"
        marketing: "Marketing"
    en:
      title: "Cookie Settings"
      # ... English text ...
```

#### `analytics` — Plausible Analytics Integration

```yaml
analytics:
  plausible:
    enabled: true                         # true/false
    baseURL: "https://analytics.company.de"  # Plausible Instance (self-hosted oder cloud)
    domain: "company.de"                  # Domain to track
    scriptPath: "/js/script.hash.outbound-links.tagged-events.js"  # Optional Script variant
```

#### `seo` — SEO & Meta-Tags

```yaml
seo:
  defaultOGImage: "/static/img/og-default.png"  # Open Graph Image (social sharing)
  defaultTwitterCard: "summary"           # "summary" oder "summary_large_image"
  globalJSONLD:                           # Global JSON-LD Schema.org
    - |
      {
        "@context":"https://schema.org",
        "@type":"Organization",
        "name":"Company",
        "url":"https://company.de",
        "email":"contact@company.de",
        "address":{"@type":"PostalAddress","addressCountry":"DE"}
      }
    - |
      {
        "@context":"https://schema.org",
        "@type":"WebSite",
        "url":"https://company.de",
        "potentialAction":{"@type":"SearchAction","target":"https://company.de/search?q={search_term}"}
      }
```

**Auto-Generated:** Webhull erstellt automatisch:
- `/sitemap.xml` (alle Pages)
- `/robots.txt` (Allow: / Disallow: /admin)
- Hreflang-Tags (für i18n)
- Canonical-Tags

#### `ui` — UI Strings & Labels (alle Texte, die nicht in Content sind)

```yaml
ui:
  de:
    # Navigation & Footer
    contactURL: "/contact"
    contactLabel: "Kontakt"
    imprintURL: "/imprint"
    imprintLabel: "Impressum"
    privacyURL: "/privacy"
    privacyLabel: "Datenschutz"
    footerTagline: "We create great things."
    allRights: "Alle Rechte vorbehalten."
    
    # Contact Form Labels & Fields
    contactForm:
      submitText: "Send Message"
      fields:
        - name: name
          label: "Dein Name"
          type: text
          required: true
          placeholder: "John Doe"
        - name: email
          label: "Deine E-Mail"
          type: email
          required: true
          placeholder: "john@example.de"
        - name: company
          label: "Unternehmen"
          type: text
          required: false
          placeholder: "ACME Corp"
        - name: subject
          label: "Betreff"
          type: text
          required: true
          placeholder: "Anfrage zu..."
        - name: message
          label: "Deine Nachricht"
          type: textarea
          required: true
          placeholder: "Schreib hier deine Nachricht..."
  
  en:
    contactURL: "/contact"
    contactLabel: "Contact"
    # ... rest in English ...
```

---

### `site/config.yaml` — Runtime Config (Helm ConfigMap)

Diese Datei wird zur **Laufzeit gemountet** (nicht ins Container-Image gebacken).

```yaml
server:
  port: "${PORT:8080}"                    # Optional | Server Port (Env-Override)
  host: "0.0.0.0"                         # Optional | Listen Address
  environment: "${ENV:production}"        # Optional | development/production
  staticDir: "static"                     # Optional | Path zu Static Files
  staticCacheMaxAge: 8760h                # Optional | Browser-Cache für Static (1 Jahr)
  read_timeout: 15s                       # Optional | HTTP Read Timeout
  write_timeout: 15s                      # Optional | HTTP Write Timeout
  idle_timeout: 60s                       # Optional | Keep-Alive Timeout
  shutdown_timeout: 30s                   # Optional | Graceful Shutdown

mail:
  host: "${SMTP_HOST:localhost}"          # Required if contact.enabled | SMTP Server
  port: ${SMTP_PORT:587}                  # Required if contact.enabled | SMTP Port
  username: "${SMTP_USER}"                # Required | SMTP Username (env var)
  password: "${SMTP_PASSWORD}"            # Required | SMTP Password (env var)
  useTLS: ${SMTP_TLS:true}                # Optional | Use TLS/SSL

gate:
  secret: "${GATE_SECRET}"                # Optional | HMAC Secret für Access-Gate
  # Wenn secret gesetzt, können Pages gated werden (Frontmatter: gate: true)

health:
  serviceVersion: "${APP_VERSION:dev}"    # Optional | Version für Health-Check
  # Health-Server läuft auf Port 8082 (standard)
```

**Env-Expansion:** `${VAR:default}` syntax wird unterstützt.

---

## 📝 CONTENT ERSTELLEN

### Content-Format: HTML-Fragmente + YAML Frontmatter

Alle Content-Dateien sind **HTML-Fragmente** (keine `<html>`, `<head>`, `<body>`, `<nav>`, `<footer>` Tags).

```html
---
id: page-id
template: default              # Template-Name (default, home, contact, legal)
title: "Page Title"
description: "Meta description for <meta name=description>"
seo_title: "Custom Title für <title> Tag"
seo_description: "Custom description für Meta-Tag"
[weitere frontmatter-felder je nach template...]
---

<!-- HTML Content hier (nur Body-Inhalt) -->
<section id="about">
  <h1>Über uns</h1>
  <p>...</p>
</section>
```

### Verfügbare Templates

| Template | Verwendung | Auto-Fields |
|----------|-----------|-------------|
| **default** | Standard-Seite | — |
| **home** | Startseite mit Hero-Section | `heroLine1`, `heroLine2`, `heroSubtitle`, `heroCTA1Text`, `heroCTA1Link`, `heroCTA2Text`, `heroCTA2Link` |
| **contact** | Contact Form Page | — (Contact Form wird auto-rendered) |
| **legal** | Rechtliche Seiten (Imprint, Privacy) | — |

### Frontmatter-Felder (Vollständige Liste)

```yaml
---
# Pflichtfelder
id: "page-slug"                    # Slug für URL (/page-slug)
title: "Page Title"                # Wird in <title>, Breadcrumb verwendet
template: default                  # Template-Name

# SEO-Felder
description: "Meta description"    # <meta name=description>
seo_title: "Custom <title> tag"   # Überschreibt title für <title> Tag
seo_description: "Custom meta"     # Überschreibt description
seo_image: "/static/img/og.png"   # OG Image (wenn anders als default)
seo_keywords: "tag1, tag2"        # Keywords (optional)

# Homepage-Template Felder
heroLine1: "First Line"
heroLine2: "Second Line"
heroSubtitle: "Subtitle text"
heroCTA1Text: "Button 1 Text"
heroCTA1Link: "/link"
heroCTA2Text: "Button 2 Text"
heroCTA2Link: "/link"
startPage: true                    # true = dies ist die Startseite (/)

# Content-Sections (optional, für Webhull-Parser)
# JSON-LD strukturierte Daten
jsonLD: |
  [{"@context":"https://schema.org",...}]

# Gate (Access Control)
gate: true                         # Seite hinter HMAC-Gate verstecken
---
```

### Beispiel: Multi-Language Content

```
site/content/
├── de/
│   ├── home.html
│   ├── about.html
│   └── contact.html
├── en/
│   ├── home.html
│   ├── about.html
│   └── contact.html
└── fr/
    ├── home.html
    ├── about.html
    └── contact.html
```

Webhull erstellt automatisch:
- `/de/` (German version)
- `/en/` (English version)
- `/fr/` (French version)
- Hreflang-Tags zwischen Versionen

### Beispiel: Single-Page mit Anchor-Navigation

```html
<!-- site/content/de/home.html -->
---
id: home
template: home
title: "ARCON"
heroLine1: "Architecture Alive"
heroLine2: "by Layer87"
---

<section id="product">
  <h2>The Product</h2>
  <p>...</p>
</section>

<section id="how-it-works">
  <h2>How It Works</h2>
  <p>...</p>
</section>

<!-- Navigation sollte dann auf diese Anchors verweisen -->
```

**`pages.yaml` Navigation (mit Anchors):**

```yaml
navigation:
  header:
    de:
      - title: "Product"
        url: "/#product"
      - title: "How It Works"
        url: "/#how-it-works"
      - title: "Contact"
        url: "/contact"
```

---

## 🔨 BUILD & DEPLOY

### Lokal Entwickeln

```bash
# Watch mode (hot-reload via docker-compose)
docker-compose up

# Browser: http://localhost:8080
# Mailpit: http://localhost:8025 (test emails)
```

### Production Build

```bash
# Build Docker Image (Webhull Binary + Site gebacken)
docker build -t my-site:1.0.0 .

# Push to Registry
docker tag my-site:1.0.0 ghcr.io/company/my-site:1.0.0
docker push ghcr.io/company/my-site:1.0.0
```

### Deploy mit Helm

```bash
# Helm Values (site-specific)
cat > values.yaml << 'EOF'
image:
  repository: ghcr.io/company/my-site
  tag: "1.0.0"

replicas: 2

config:
  server:
    port: 8080
  mail:
    host: "mail.company.de"
    port: 465
    useTLS: true

secrets:
  smtp_username: "user@company.de"
  smtp_password: "secret"

ingress:
  enabled: true
  host: "company.de"
  cert: "letsencrypt-prod"
EOF

# Deploy
helm install my-site ./deploy/helm -f values.yaml -n websites
```

### Output-Struktur

Webhull kompiliert zu einem **Single Binary**:
- `/app/webhull` — executable
- `/app/site/` — gesamte Site gebacken rein (config.yaml wird zur Runtime gemountet)
- `/app/site/static/` — CSS, JS, Images (Cache-busted)

**Es gibt kein Build-Output-Verzeichnis.** Webhull rendert direkt HTTP-Responses.

---

## 📋 HÄUFIGE MUSTER (Copy-Paste Templates)

### Pattern 1: Multi-Language B2B Site (wie Layer87 Website)

```yaml
# pages.yaml
site:
  name: "Layer87"
  baseURL: "https://layer87.de"
  copyrightStartYear: 2025

i18n:
  defaultLanguage: "de"
  languages: ["de", "en"]

navigation:
  header:
    de:
      - title: "Leistungen"
        url: "/leistungen"
      - title: "Produkte"
        url: "/produkte"
      - title: "Über uns"
        url: "/ueber-uns"
    en:
      - title: "Services"
        url: "/services"
      - title: "Products"
        url: "/products"
      - title: "About"
        url: "/about"

contact:
  enabled: true
  recipientName: "Sales"
  recipients:
    - "contact@company.de"
  subject:
    de: "[Anfrage] {{.Name}}"
    en: "[Inquiry] {{.Name}}"

ui:
  de:
    contactForm:
      fields:
        - name: name
          label: "Ihr Name"
          type: text
          required: true
        - name: email
          label: "E-Mail"
          type: email
          required: true
        - name: company
          label: "Unternehmen"
          type: text
        - name: message
          label: "Nachricht"
          type: textarea
          required: true
  en:
    contactForm:
      fields:
        - name: name
          label: "Your Name"
          type: text
          required: true
        - name: email
          label: "Email"
          type: email
          required: true
        - name: company
          label: "Company"
          type: text
        - name: message
          label: "Message"
          type: textarea
          required: true
```

### Pattern 2: Single-Page Product Launch (wie ARCON)

```yaml
# pages.yaml
site:
  name: "ARCON"
  baseURL: "https://arcon.layer87.de"

i18n:
  defaultLanguage: "de"
  languages: ["de"]

navigation:
  header:
    de:
      - title: "Product"
        url: "/#product"
      - title: "Magic"
        url: "/#magic"
      - title: "Use Cases"
        url: "/#use-cases"
      - title: "Early Access"
        url: "/#access"

contact:
  enabled: true
  recipientName: "ARCON Team"
  recipients:
    - "hello@layer87.de"
  subject:
    de: "[ARCON] Early Access anfrage"

ui:
  de:
    contactForm:
      fields:
        - name: name
          label: "Name"
          type: text
          required: true
        - name: email
          label: "Email"
          type: email
          required: true
```

**HTML Content:**

```html
<!-- site/content/de/home.html -->
---
id: home
template: home
title: "ARCON"
heroLine1: "Architecture Alive"
heroLine2: "by Layer87"
---

<section id="product">
  <h2>The Product</h2>
  ...
</section>

<section id="magic">
  <h2>So funktioniert es</h2>
  ...
</section>

<section id="access">
  <h2>Early Access</h2>
  ...
</section>
```

### Pattern 3: Creative Portfolio (wie Studio OptiMayS)

```yaml
# pages.yaml
site:
  name: "Studio OptiMayS"
  baseURL: "https://studio-optimays.de"

navigation:
  header:
    de:
      - title: "The Visionary"
        url: "/#visionary"
      - title: "The Studio"
        url: "/#studio"
      - title: "The Work"
        url: "/#work"
      - title: "The Contact"
        url: "/#contact"

contact:
  enabled: true
  recipients:
    - "hello@studio-optimays.de"
  subject:
    de: "[Contact] Neue Nachricht"

consent:
  enabled: true
  categories:
    necessary:
      required: true
      default: true
    analytics:
      required: false
      default: false
```

### Pattern 4: Internal B2B Portal (wie Layer87 Marketplace)

```yaml
# pages.yaml
site:
  name: "Layer87 Marketplace"
  baseURL: "https://marketplace.layer87.de"

navigation:
  header:
    de:
      - title: "Services"
        url: "/services"
      - title: "Pricing"
        url: "/pricing"

contact:
  enabled: false  # Kein Contact Form, Mail-Link stattdessen

consent:
  enabled: false  # Keine Cookies nötig (internal site)

analytics:
  plausible:
    enabled: false  # Optional: disabled
```

### Pattern 5: Newsletter Signup (Extra)

Webhull hat keinen nativen Newsletter-Feature, aber man kann Contact Form für Signup verwenden:

```yaml
contact:
  enabled: true
  recipientName: "Newsletter"
  recipients:
    - "newsletter@company.de"
  subject:
    de: "[Newsletter Signup] {{.Name}}"

ui:
  de:
    contactForm:
      fields:
        - name: email
          label: "E-Mail-Adresse"
          type: email
          required: true
        - name: name
          label: "Name"
          type: text
          required: false
      submitText: "Subscribe"
```

---

## ✅ DO's

- ✅ **Nutze HTML-Fragmente** für Content (kein vollständiges HTML)
- ✅ **YAML Frontmatter** für alle Metadaten
- ✅ **CSS-Klassen** für Styling (keine Inline-Styles wenn möglich)
- ✅ **Self-hosted Fonts** (keine Google Fonts wegen Privacy)
- ✅ **Environment-Variablen** in `config.yaml` für Secrets (`${SMTP_HOST}`)
- ✅ **Multi-language structure** vorbereiten, auch wenn zunächst nur 1 Sprache
- ✅ **Semantic HTML** (Section, Article, Header, Footer Elements)
- ✅ **JSON-LD Schema.org** für bessere SEO
- ✅ **Webhull Templates** verwenden (home, contact, legal)
- ✅ **Cache Headers** in Production nutzen (static 1 Jahr via cache-busting)

---

## ❌ DON'Ts

- ❌ **Webhull Framework nicht verändern** — nur nutzen, nicht extenden
- ❌ **Keine neue CSS-Architektur erfinden** — verwende bestehende Patterns
- ❌ **Keine Database** integrieren (Webhull ist stateless)
- ❌ **Keine User Authentication** außer einfachem HMAC-Gate
- ❌ **Keine Node-Dependencies** — kein npm/webpack
- ❌ **Keine Frontend-Frameworks** (Vue, React, etc.)
- ❌ **Keine Runtime-Generated Content** — alles muss statisch sein
- ❌ **config.yaml nicht ins Image backen** — nur pages.yaml!
- ❌ **Keine Hardcoded Secrets** — immer Env-Variablen nutzen
- ❌ **Keine SPA-Routing** — Webhull ist SSR nur

---

## 🔍 DEBUGGING

### Häufige Fehler

| Fehler | Ursache | Lösung |
|--------|--------|--------|
| "Page not found" | Content-Datei existiert nicht | Check: `site/content/de/[slug].html` |
| Contact Form funktioniert nicht | SMTP nicht konfiguriert | Check: `config.yaml` mail.host, -port |
| Bestätigungs-Mail kommt nicht an | bodyFile-Pfad falsch | Check: `mail/de-confirmation.html` existiert |
| Navigation nicht sichtbar | Slug in pages.yaml falsch | Slug muss Dateinamen ohne `.html` sein |
| i18n funktioniert nicht | Sprache nicht in `languages` | Add zu `i18n.languages: ["de", "en"]` |
| Static Files laden nicht | staticDir-Pfad falsch | Default: `static/` im Site-Root |

### Log-Ausgaben

```bash
# Docker Logs ansehen
docker-compose logs webhull

# Fehlersuche
docker-compose logs webhull | grep -i error
docker-compose logs webhull | grep -i smtp
```

---

## 📚 WEITERE RESSOURCEN

- **Webhull Repository:** https://github.com/layer87-labs/webhull
- **AGENTS.md:** Guidance für KI-Coding-Agents
- **Example Repos:**
  - Layer87 Website: Multi-Page B2B
  - ARCON: Single-Page Product
  - Studio OptiMayS: Portfolio
  - Layer87 Marketplace: Internal Portal

---

## Kurz-Referenz: Was Agent Zwingend Wissen Muss

1. **Zwei Config-Dateien:** `pages.yaml` (baked in) + `config.yaml` (runtime)
2. **HTML-Fragmente:** Keine `<html>`, `<body>`, `<nav>`, `<footer>` Tags
3. **YAML Frontmatter:** Alle Metadaten dort
4. **Language Paths:** `/de/`, `/en/`, etc. (basierend auf `i18n.languages`)
5. **Templates:** default, home, contact, legal
6. **Single Binary:** Webhull kompiliert alles zu einem Executable
7. **No Database:** Webhull ist stateless
8. **Env-Expansion:** `${VAR:default}` in config.yaml
9. **Docker:** Containerfile mit `FROM ghcr.io/layer87-labs/webhull:1.2.1`
10. **Deployment:** Helm ConfigMaps für Runtime-Config

---

**Ende des Webhull Skill.md**

Generated: 2026-05-30  
Framework Version: 1.2.1  
Referenz-Sites: 4 (Layer87, ARCON, OptiMayS, Marketplace)
