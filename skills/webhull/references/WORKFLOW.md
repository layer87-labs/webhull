# Agent Workflow

Step-by-step guide from empty directory to deployed Webhull website.

---

## Phase 1 — Clarify Requirements (5 min)

Before writing any files, determine:

1. **Site type** — Multi-page B2B / single-page landing / portfolio / internal portal?
2. **Languages** — Single language or multilingual? Which language codes?
3. **Features** — Contact form? Analytics? Cookie consent? Access gate?
4. **Deployment** — Docker only, or Helm + Kubernetes?

→ Pick a pattern from [PATTERNS.md](PATTERNS.md).

---

## Phase 2 — Create Structure (2 min)

```bash
mkdir -p site/{content/de,mail,static/{css,fonts,js,img}}
```

For multilingual:

```bash
mkdir -p site/content/{de,en}
```

---

## Phase 3 — Write Config Files (10 min)

### site/pages.yaml

Copy the matching pattern from [PATTERNS.md](PATTERNS.md), then adjust:

- `site.name`, `site.baseURL`
- `i18n.languages`
- `navigation.header` / `navigation.footer`
- `contact.recipients`, `contact.subject`
- `ui.[lang].footerTagline`, `ui.[lang].contactForm.fields`
- Enable/disable `consent`, `analytics`, `gate` as needed

### site/config.yaml

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

No secrets. No hardcoded values. All via `${VAR:default}`.

---

## Phase 4 — Write Content Files (15 min)

For each page:

1. Create `site/content/[lang]/[slug].html`
2. Add YAML frontmatter (`id`, `template`, `title`, SEO fields)
3. Write HTML body — fragments only, no `<html>` / `<body>` / `<nav>` / `<footer>`

**Minimum pages for a functional site:**

| File | Template | Notes |
|------|----------|-------|
| `home.html` | `home` | Set `startPage: true` |
| `contact.html` | `contact` | Only if `contact.enabled: true` |
| `imprint.html` | `legal` | Required in Germany/EU |
| `privacy.html` | `legal` | Required in Germany/EU |

**Contact form confirmation email:**

Create `site/mail/de-confirmation.html` (see [PATTERNS.md](PATTERNS.md) for template).

---

## Phase 5 — Add Static Assets (10 min)

- `site/static/css/style.css` — full design system
- `site/static/fonts/` — self-hosted WOFF2 files + `fonts.css`
- `site/static/img/` — logo, favicon, OG image
- `site/static/js/` — optional vanilla JS

No CDN links. No Google Fonts. No npm.

---

## Phase 6 — Build & Test Locally (5 min)

```bash
# Containerfile
echo 'FROM ghcr.io/layer87-labs/webhull:1.2.1
COPY site /app/site' > Containerfile

# docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: "3.8"
services:
  webhull:
    build: .
    ports: ["8080:8080"]
    volumes: ["./site:/app/site"]
    environment:
      PORT: "8080"
      SMTP_HOST: "mailpit"
      SMTP_PORT: "1025"
      SMTP_TLS: "false"
  mailpit:
    image: axllent/mailpit:latest
    ports: ["8025:8025", "1025:1025"]
EOF

docker-compose up --build
```

**Verify:**

- [ ] `http://localhost:8080` — homepage loads
- [ ] Navigation visible, active state correct
- [ ] All pages accessible
- [ ] Contact form renders and submits
- [ ] Confirmation email appears in Mailpit (`http://localhost:8025`)
- [ ] No errors in `docker-compose logs webhull`

---

## Phase 7 — Deploy (10 min)

### Build & push image

```bash
docker build -t my-site:1.0.0 .
docker tag my-site:1.0.0 ghcr.io/org/my-site:1.0.0
docker push ghcr.io/org/my-site:1.0.0
```

### Helm values

```yaml
image:
  repository: ghcr.io/org/my-site
  tag: "1.0.0"

replicas: 2

config:
  server:
    port: 8080
    environment: production
  mail:
    host: "mail.company.de"
    port: 465
    useTLS: true

secrets:
  smtp_user: "${SMTP_USER}"
  smtp_password: "${SMTP_PASSWORD}"

ingress:
  enabled: true
  host: "my-site.de"
```

```bash
helm install my-site ./deploy/helm -f values.yaml -n websites
```

### Verify deployment

```bash
kubectl get pods -n websites
kubectl logs -n websites -l app=my-site
curl https://my-site.de/
```

---

## Decision Tree

```
"Build a website"
      │
      ├── Many pages (>4), multi-nav?
      │       └── Pattern 1 (Multi-Page B2B)
      │
      ├── One main page, anchor nav?
      │       ├── Product launch / waitlist? → Pattern 2 (Product Launch)
      │       └── Creative / portfolio?      → Pattern 3 (Portfolio)
      │
      └── Internal tool, no public tracking?
              └── Pattern 4 (Internal Portal)
```

---

## Time Budget

| Phase | Task | ~Time |
|-------|------|-------|
| 1 | Clarify requirements | 5 min |
| 2 | Create structure | 2 min |
| 3 | Write config files | 10 min |
| 4 | Write content files | 15 min |
| 5 | Add static assets | 10 min |
| 6 | Build & test locally | 5 min |
| 7 | Deploy | 10 min |
| | **Total** | **~60 min** |
