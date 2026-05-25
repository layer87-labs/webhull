---
sidebar_position: 3
title: Access Gate
---

# Access Gate

Optional code-based protection — either for the entire site (`gate`) or only for a specific path prefix (`arconGate`). Visitors see a minimal login page, enter a code, and receive a signed session cookie on success. No external auth system, no admin UI, no database.

Both modes can be active simultaneously and run fully independently (separate cookies, separate code lists, separate rate limiters).

## How it works

```
Visitor → GET /any-page
        → no / invalid cookie → 302 /gate?redirect=/any-page
        → Gate page (logo + code input)
        → POST /gate { code: "aurora-pine-7" }
        → ✅ valid   → set session cookie → 302 /any-page
        → ❌ invalid → gate page with error (rate limit: 3 attempts / 15 min / IP)
```

Always public: `/gate`, `/static/*`, `/health`, `/sitemap.xml`, `/robots.txt`

## Configuration

```yaml
gate:
  enabled: true
  cookieSecret: "${GATE_SECRET}"   # HMAC key — inject via env var; required
  cookieName: "gate_session"       # optional, default: gate_session
  cookieMaxAge: 24h                # optional, default: 24h
  codes:
    - code: "aurora-pine-7"
      label: "Prospect — Acme Corp"
    - code: "silver-lake-3"
      label: "Partner XYZ"
  # Optional: override error messages (useful for non-English sites)
  msgTooManyAttempts: "Too many attempts. Please try again later."
  msgInvalidCode: "Invalid access code. Please try again."
```

Without a `gate:` block (or `enabled: false`): the site is fully public, zero overhead.

### Generate a secret

```bash
openssl rand -base64 32
```

Minimum length: 32 characters. Use a unique secret per deployment environment.

### Choosing codes

Codes are plaintext strings — no hashing, no database. Recommendation: 3 random words + a digit, e.g. `cedar-frost-9`. Use one code per recipient so individual codes can be rotated independently.

## ArconGate — path-scoped protection

Protects only `/arcon/*`. The rest of the site stays fully public. Ideal for product demos or gated experience areas accessible only to selected visitors.

```
Visitor → GET /arcon/
        → no / invalid cookie → 302 /arcon/gate?redirect=/arcon/
        → Gate page (code input)
        → POST /arcon/gate { code: "cedar-frost-9" }
        → ✅ valid   → arcon_session cookie → 302 /arcon/
        → ❌ invalid → gate page with error (rate limit: 3 attempts / 15 min / IP)
```

Always public: `/arcon/gate`, everything outside `/arcon/*`.

```yaml
arconGate:
  enabled: true
  contentDir: "arcon"                    # directory of HTML files, relative to site.yaml
  cookieSecret: "${ARCON_GATE_SECRET}"   # HMAC key — inject via env var
  cookieName: "arcon_session"            # optional, default: arcon_session
  cookieMaxAge: 8h                       # optional, default: 8h
  title: "My Product – Access"           # gate page title, default: "<site.name> – Access"
  codes:
    - code: "cedar-frost-9"
      label: "Demo prospect"
```

Files in `contentDir` are served statically at `/arcon/<file>.html`.

## Local testing

```bash
GATE_SECRET="local-dev-secret-min32chars!!" \
PORT=8081 \
./build/bin/webhull -config example-gate/site.yaml
```

The `example-gate/` directory shows a complete gate configuration.

## Kubernetes / Helm deployment

### Create a Kubernetes Secret

```bash
kubectl create secret generic webhull-gate-secret \
  --from-literal=secret="$(openssl rand -base64 32)" \
  --namespace my-namespace
```

Rotate the secret (invalidates all active sessions):

```bash
kubectl patch secret webhull-gate-secret \
  --namespace my-namespace \
  --type=merge \
  -p '{"stringData":{"secret":"'$(openssl rand -base64 32)'"}}'
```

### Inject via Helm values

```yaml
frontend:
  env:
    - name: GATE_SECRET
      valueFrom:
        secretKeyRef:
          name: webhull-gate-secret
          key: secret
```

## Security properties

| Property | Details |
|----------|---------|
| Cookie signing | HMAC-SHA256, Go stdlib `crypto/hmac` |
| Timing attacks | `crypto/subtle.ConstantTimeCompare` for all code comparisons |
| Cookie flags | `HttpOnly`, `Secure`, `SameSite=Strict` |
| Session expiry | Verified server-side (timestamp in cookie payload) |
| Rate limiting | 3 attempts / 15 min / IP, in-memory |
| No JWT | No external auth library — Go stdlib only |

**Not protected against:** authorised users sharing their code. For sensitive content, rotate codes regularly.
