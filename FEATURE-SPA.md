# Feature: Single-Page Application (SPA) Support

Branch: `feature/spa-single-page-app`

## Problem

Webhull derzeit ist **multi-page-focused**:
- Routes wie `/home`, `/about`, `/contact` → separate HTML pages
- Navigation ist page-based (header links zu verschiedenen URLs)

Aber viele Sites sind **Single-Page Apps** mit **Anchor-Navigation**:
- Eine großartig große HTML-Datei mit vielen `<section id="...">` Blöcke
- Navigation via `<a href="#section">` (smooth scroll)
- Kein Page-Reload
- Multi-language: `/` (de), `/en/` (en), usw. — aber jeweils SPA-Pattern

**Beispiel: Studio OptiMayS**
```
GET / → SPA (de)
  └── <section id="hero">
  └── <section id="visionary">
  └── <section id="studio">
  └── <section id="magic">
  └── <section id="contact">
GET /impressum → separate Page (legal)
GET /datenschutz → separate Page (legal)
```

## Solution: SPA-Type in Pages.yaml

### Config-Pattern

```yaml
site:
  name: "Studio OptiMayS"
  type: "spa"  # ← NEW: declares this is a single-page-app
  baseURL: "https://studio-optimays.de"
  ...

i18n:
  defaultLanguage: "de"
  languages: ["de", "en"]

pages:
  # SPA root (auto-rendered from content/de/index.html or home.html)
  - id: "home"
    i18n:
      de:
        slug: ""  # ← Empty slug = root (GET /)
        title: "Studio OptiMayS"
        description: "..."
      en:
        slug: "en"  # ← /en/
        title: "Studio OptiMayS"
        description: "..."

  # Separate pages (legal)
  - id: "impressum"
    i18n:
      de:
        slug: "impressum"
        title: "Impressum"
      en:
        slug: "imprint"
        title: "Imprint"

# Navigation: anchor-based (not page-based)
navigation:
  header:
    de:
      - title: "The Visionary"
        url: "#visionary"  # ← Anchor, not /page
      - title: "The Studio"
        url: "#studio"
      - title: "The Contact"
        url: "#contact"
```

### Webhull Behavior

When `site.type == "spa"`:

1. **Root route** (empty slug) → GET `/` or GET `/en/` (depending on language)
2. **Anchor navigation** → NO page reload on link clicks
3. **Header/Footer** → Still rendered by webhull (layout wrapper)
4. **Content file** → All sections in one file (`content/de/index.html`)
5. **Other pages** → Still work normally (impressum, datenschutz)

### URL Routing

**Multi-language SPA:**
```
GET /        → renders home.html (de)
GET /en      → renders home.html (en)
GET /en/     → (alias for /en)
GET /impressum  → renders impressum.html (de)
GET /en/imprint → renders imprint.html (en)
```

**Single-language SPA:**
```
GET /        → renders home.html
GET /impressum → renders impressum.html
```

## Implementation Checklist

### Phase 1: Config Support
- [ ] Update `internal/pkg/pages/models.go`: add `site.Type` field
- [ ] Update `internal/pkg/config/loader.go`: parse `site.type` from pages.yaml
- [ ] Update `internal/pkg/pages/service.go`: handle empty slug for root routes

### Phase 2: Routing
- [ ] Update `internal/app/server/server.go`: route handling for SPA
  - GET `/` → home (de)
  - GET `/en` or GET `/en/` → home (en)
  - Preserve existing page routes for legal pages
- [ ] Anchor links should NOT trigger page navigation (CSS/JS handles it)

### Phase 3: Templates
- [ ] Keep existing templates (home, default, legal, notfound)
- [ ] Ensure smooth scroll works (CSS: `html { scroll-behavior: smooth; }`)
- [ ] No changes needed — content drives the structure

### Phase 4: Documentation
- [ ] Update AGENTS.md with SPA pattern
- [ ] Add example config for SPA sites
- [ ] Document anchor-based navigation

## Testing

**Test Site: Studio OptiMayS**

```bash
# Clone optimays-website on main
git clone ...optimays-website

# Configure for SPA (pages.yaml with site.type: "spa")
# Content: single home.html with all sections

# Run webhull
webhull -config config.yaml -pages pages.yaml

# Test:
curl http://localhost:8080/             # 200 OK (SPA de)
curl http://localhost:8080/en           # 200 OK (SPA en)
curl http://localhost:8080/impressum    # 200 OK (legal page)
```

## Benefits

✅ **Clean separation**: SPA vs Multi-page sites  
✅ **Natural routing**: `/` instead of `/home`  
✅ **Multi-language**: Each language has its own SPA  
✅ **No template changes**: Existing templates work fine  
✅ **Scalable**: Can be extended for future patterns  

## Notes

- SPA content is ONE file per language (e.g., `content/de/index.html`, `content/en/index.html`)
- All sections in one file = no page reload needed
- JS/CSS handles smooth scroll + active nav state (not webull)
- Legal pages are separate (impressum, datenschutz) — webhull handles normally
