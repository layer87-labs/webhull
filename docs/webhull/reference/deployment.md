---
sidebar_position: 4
title: Deployment
---

# Deployment

## Container image

webhull is distributed as a container image built on [Chainguard's distroless static base](https://github.com/chainguard-images/images/tree/main/images/static). The image contains only the compiled binary — no shell, no package manager, no OS utilities.

```dockerfile
FROM cr.layer87.de/layer87/webhull:1.2.3

# Copy your site structure and content into the image
COPY site/pages.yaml /app/pages.yaml
COPY site/content /app/content
COPY site/static /app/static
```

The binary inside the image expects:
- Config file mounted at runtime (typically as a Kubernetes ConfigMap)
- Pages/content baked into the image at build time

### Containerfile example

```dockerfile
FROM cr.layer87.de/layer87/webhull:1.2.3

WORKDIR /app

COPY site/pages.yaml .
COPY site/content ./content
COPY site/static ./static
```

### Running locally

```bash
docker run --rm \
  -p 8080:8080 \
  -v $(pwd)/deploy/config.yaml:/app/config.yaml \
  -e GATE_SECRET="dev-secret-32-chars-minimum!!" \
  -e ENVIRONMENT=production \
  my-site:latest \
  /bin/webhull -config /app/config.yaml -pages /app/pages.yaml
```

---

## Split config model

In production, keep operational config separate from site content:

```
deploy/
  config.yaml     ← Kubernetes ConfigMap — server, mail, analytics, gate
  Containerfile

site/
  pages.yaml      ← baked into image — identity, i18n, navigation, ui, contentDir
  content/
    en/
      home.html
  static/
    css/
    img/
```

**config.yaml** (mounted as ConfigMap):
```yaml
site:
  name: "${SITE_NAME:My Site}"
  baseURL: "${BASE_URL}"

server:
  port: "${PORT:8080}"
  environment: "${ENVIRONMENT:production}"

mail:
  host: "${SMTP_HOST}"
  password: "${SMTP_PASSWORD}"

gate:
  enabled: true
  cookieSecret: "${GATE_SECRET}"
  codes:
    - code: "${GATE_CODE}"
      label: "main"
```

**pages.yaml** (baked into image):
```yaml
contentDir: "content"

i18n:
  defaultLanguage: "en"
  languages: ["en"]

navigation:
  header:
    en:
      - title: "About"
        url: "/about"
        slug: "about"
```

---

## Helm chart

The `deploy/helm/` directory contains a Helm chart for Kubernetes deployment.

### Install

```bash
helm install my-site deploy/helm/ \
  --namespace my-namespace \
  --set frontend.image.tag=1.2.3
```

### Key values

```yaml
frontend:
  enabled: true
  image:
    repository: cr.layer87.de/layer87/webhull
    tag: "1.2.3"
    pullPolicy: Always
  replicas: 1
  service:
    type: ClusterIP
    port: 80
    targetPort: 8080
  env:
    - name: ENVIRONMENT
      value: "production"
    - name: BASE_URL
      value: "https://example.com"
    - name: GATE_SECRET
      valueFrom:
        secretKeyRef:
          name: webhull-gate-secret
          key: secret

# Operational config (rendered as a ConfigMap, mounted at /app/config/config.yaml)
backend:
  config:
    enabled: true
    data:
      server:
        port: "8080"
        environment: "${ENVIRONMENT:production}"

# Gateway API ingress
ingress:
  enabled: true
  type: gateway
  gatewayClassName: istio
  hosts:
    - host: example.com
  tls:
    - secretName: example-tls
      hosts:
        - example.com

# Health server for Kubernetes probes
health:
  enabled: true
  port: 9090

# Autoscaling
autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 3
```

---

## Health probes

When `health.enabled: true`, the health server runs on a dedicated port with three endpoints:

| Endpoint | Use | Response |
|----------|-----|----------|
| `GET /health` | Kubernetes liveness probe | `{"status":"ok"}` |
| `GET /ready` | Kubernetes readiness probe | `{"status":"ready"}` |
| `GET /metrics` | Prometheus scrape | `webhull_info`, `webhull_uptime_seconds` |

Kubernetes probe configuration:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 9090
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 9090
  initialDelaySeconds: 3
  periodSeconds: 5
```

---

## Production checklist

- [ ] `server.environment: "production"` — enables release mode and JSON logging
- [ ] `site.baseURL` set to the public HTTPS URL (no trailing slash)
- [ ] `GATE_SECRET` injected via Kubernetes Secret, not hardcoded
- [ ] `SMTP_PASSWORD` injected via Kubernetes Secret
- [ ] `seo_noindex: true` on any staging / preview environments (or use a separate gate)
- [ ] Health server enabled and wired to Kubernetes liveness/readiness probes
- [ ] `server.staticCacheMaxAge: 8760h` — long cache for static assets (cache-busted via content hash)
- [ ] Container image built with specific version tag, not `latest`
- [ ] `webhull -validate` run in CI before deploying
