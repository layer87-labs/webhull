---
sidebar_position: 2
title: Environment Variables
---

# Environment Variables

webhull expands `${VAR:default}` placeholders throughout the YAML config at load time. This page lists every variable referenced in the example configs and what it maps to.

## Syntax

| Syntax | Behaviour |
|--------|-----------|
| `${VAR}` | Value of `VAR`; empty string if unset |
| `${VAR:default}` | Value of `VAR`; falls back to `default` if unset or empty |
| `$VAR` | Value of `VAR`; empty string if unset (no default possible) |

Expansion happens before YAML parsing — the binary never sees raw variable names in production.

## Variables reference

| Variable | Used in | Default | Description |
|----------|---------|---------|-------------|
| `PORT` | `server.port` | `8080` | HTTP listen port. |
| `HOST` | `server.host` | `0.0.0.0` | HTTP listen address. |
| `ENVIRONMENT` | `server.environment` | `development` | Runtime mode. Set to `production` in deployed environments. |
| `BASE_URL` | `site.baseURL` | — | Absolute base URL, e.g. `https://example.com`. No trailing slash. |
| `SMTP_HOST` | `mail.host` | — | SMTP server hostname. |
| `SMTP_PORT` | `mail.port` | `587` | SMTP port. |
| `SMTP_USERNAME` | `mail.username` | — | SMTP auth username. |
| `SMTP_PASSWORD` | `mail.password` | — | SMTP auth password. |
| `SMTP_FROM` | `mail.from` | — | Sender email address. |
| `GATE_SECRET` | `gate.cookieSecret` | — | HMAC-SHA256 signing key for the site gate. Min 32 chars. **Never hardcode.** |
| `ARCON_GATE_SECRET` | `arconGate.cookieSecret` | — | HMAC-SHA256 signing key for the arcon gate. Min 32 chars. **Never hardcode.** |
| `PLAUSIBLE_BASE_URL` | `analytics.plausible.baseURL` | — | Plausible server URL (self-hosted or `https://plausible.io`). |
| `PLAUSIBLE_DOMAIN` | `analytics.plausible.domain` | — | Plausible site domain for `data-domain` attribute. |
| `COLLECTOR_ENDPOINT` | `analytics.collector.endpoint` | — | Custom analytics collector HTTP endpoint. |

## Generating secrets

```bash
# Generate a gate secret (32 bytes, base64-encoded)
openssl rand -base64 32
```

## Kubernetes Secret example

```bash
kubectl create secret generic webhull-secrets \
  --from-literal=gate-secret="$(openssl rand -base64 32)" \
  --from-literal=smtp-password="changeme" \
  --namespace my-namespace
```

```yaml
# Helm values — inject into the container
frontend:
  env:
    - name: GATE_SECRET
      valueFrom:
        secretKeyRef:
          name: webhull-secrets
          key: gate-secret
    - name: SMTP_PASSWORD
      valueFrom:
        secretKeyRef:
          name: webhull-secrets
          key: smtp-password
    - name: ENVIRONMENT
      value: "production"
    - name: BASE_URL
      value: "https://example.com"
```
