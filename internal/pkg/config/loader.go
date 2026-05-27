package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a site configuration from one or two YAML files.
// When pagesPath is non-empty, the config is split:
//   - configPath: operational settings (server, mail, analytics, consent, seo, gate, health)
//   - pagesPath:  site identity, i18n, navigation, ui strings, contentDir
//
// When pagesPath is empty, configPath is treated as a monolithic site.yaml (backwards-compatible).
// Pages content is NOT loaded here — use content.Load() separately to populate Pages.
func Load(configPath string, pagesPath string) (*SiteConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	cfg, err := parseRaw(data)
	if err != nil {
		return nil, fmt.Errorf("config %s: %w", configPath, err)
	}

	// Merge pages file if provided
	if pagesPath != "" {
		pagesData, err := os.ReadFile(pagesPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read pages file %s: %w", pagesPath, err)
		}
		if err := mergePages(cfg, pagesData); err != nil {
			return nil, fmt.Errorf("pages %s: %w", pagesPath, err)
		}
	}

	// Resolve paths relative to the file that defines them.
	// Content-related paths resolve relative to pagesPath (or configPath if monolithic).
	contentBase := configPath
	if pagesPath != "" {
		contentBase = pagesPath
	}
	contentDir := filepath.Dir(contentBase)

	// Auto-detect staticDir: if not configured, look for a "static/" directory
	// next to the pages file (or config file in monolithic mode). This covers the
	// common case — COPY site/static/ site/static/ in a Containerfile — with zero
	// config required from the operator.
	if cfg.Server.StaticDir == "" {
		candidate := filepath.Join(contentDir, "static")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			cfg.Server.StaticDir = candidate // already absolute — skip Join below
		}
	}

	// Resolve relative paths to absolute using the directory that owns the file.
	// Absolute paths are used as-is (filepath.Join would corrupt them on non-Unix
	// systems or produce double-segments like /app/site/app/site/static).
	if cfg.ContentDir != "" && !filepath.IsAbs(cfg.ContentDir) {
		cfg.ContentDir = filepath.Join(contentDir, cfg.ContentDir)
	}
	if cfg.Server.StaticDir != "" && !filepath.IsAbs(cfg.Server.StaticDir) {
		cfg.Server.StaticDir = filepath.Join(contentDir, cfg.Server.StaticDir)
	}
	if cfg.ArconGate.ContentDir != "" && !filepath.IsAbs(cfg.ArconGate.ContentDir) {
		cfg.ArconGate.ContentDir = filepath.Join(contentDir, cfg.ArconGate.ContentDir)
	}

	// Resolve mail template body files.
	// In split-config mode (pagesPath != ""), templates are typically bundled
	// with the site content, so resolve relative to pagesPath directory.
	// In monolithic mode, resolve relative to configPath directory.
	// Absolute paths are always used as-is.
	templateBase := filepath.Dir(configPath)
	if pagesPath != "" {
		templateBase = filepath.Dir(pagesPath)
	}
	for lang, tmpl := range cfg.Mail.Templates {
		if tmpl.BodyFile != "" {
			absPath := tmpl.BodyFile
			if !filepath.IsAbs(absPath) {
				absPath = filepath.Join(templateBase, absPath)
			}
			content, err := os.ReadFile(absPath)
			if err != nil {
				return nil, fmt.Errorf("mail template bodyFile for %q: %w", lang, err)
			}
			tmpl.Body = string(content)
			tmpl.BodyFile = ""
			cfg.Mail.Templates[lang] = tmpl
		}
	}

	// Derive defaults from site identity (DRY: defined once)
	applyDefaults(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// applyDefaults fills empty fields with sensible defaults derived from site identity.
// All defaults are only applied when the field is still at zero-value — explicit
// configuration (from config.yaml or pages.yaml) always takes precedence.
func applyDefaults(cfg *SiteConfig) {
	// ── Server defaults ──────────────────────────────────────────────────────
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Environment == "" {
		cfg.Server.Environment = "production"
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 15 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 15 * time.Second
	}
	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = 60 * time.Second
	}
	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = 30 * time.Second
	}
	if cfg.Server.CacheMaxAge == 0 {
		cfg.Server.CacheMaxAge = 1 * time.Hour
	}
	if cfg.Server.StaticCacheMaxAge == 0 {
		cfg.Server.StaticCacheMaxAge = 8760 * time.Hour // 1 year
	}

	// ── Health defaults ──────────────────────────────────────────────────────
	// Health is enabled by default. To disable, set health.enabled: false explicitly.
	if cfg.Health.Enabled == nil {
		t := true
		cfg.Health.Enabled = &t
	}
	if cfg.Health.Host == "" {
		cfg.Health.Host = "0.0.0.0"
	}
	if cfg.Health.Port == 0 {
		cfg.Health.Port = 8082
	}
	if cfg.Health.HealthPath == "" {
		cfg.Health.HealthPath = "/health"
	}
	if cfg.Health.ReadyPath == "" {
		cfg.Health.ReadyPath = "/ready"
	}
	if cfg.Health.MetricsPath == "" {
		cfg.Health.MetricsPath = "/metrics"
	}
	if cfg.Health.Timeout == 0 {
		cfg.Health.Timeout = 5 * time.Second
	}

	// ── I18n defaults ────────────────────────────────────────────────────────
	if cfg.I18n.DefaultLanguage == "" {
		cfg.I18n.DefaultLanguage = "de"
	}
	if len(cfg.I18n.Languages) == 0 {
		cfg.I18n.Languages = []string{cfg.I18n.DefaultLanguage}
	}

	// ── Site defaults ────────────────────────────────────────────────────────
	if cfg.Site.CopyrightStartYear == 0 {
		cfg.Site.CopyrightStartYear = time.Now().Year()
	}

	// ── Contact defaults ─────────────────────────────────────────────────────
	if cfg.Contact.MaxLinks == 0 {
		cfg.Contact.MaxLinks = 2
	}
	if cfg.Contact.RateLimit.Requests == 0 {
		cfg.Contact.RateLimit.Requests = 3
	}
	if cfg.Contact.RateLimit.Window == 0 {
		cfg.Contact.RateLimit.Window = 15 * time.Minute
	}

	// ── Mail defaults ────────────────────────────────────────────────────────
	if cfg.Mail.Port == 0 {
		cfg.Mail.Port = 587
	}

	// ── Gate defaults ────────────────────────────────────────────────────────
	if cfg.Gate.CookieName == "" {
		cfg.Gate.CookieName = "gate_session"
	}
	if cfg.Gate.CookieMaxAge == 0 {
		cfg.Gate.CookieMaxAge = 24 * time.Hour
	}
	if cfg.ArconGate.CookieName == "" {
		cfg.ArconGate.CookieName = "arcon_session"
	}
	if cfg.ArconGate.CookieMaxAge == 0 {
		cfg.ArconGate.CookieMaxAge = 8 * time.Hour
	}
	if cfg.ArconGate.Title == "" {
		if cfg.Site.Name != "" {
			cfg.ArconGate.Title = cfg.Site.Name + " – Access"
		} else {
			cfg.ArconGate.Title = "Access"
		}
	}

	// ── Domain-derived defaults (DRY) ────────────────────────────────────────
	if cfg.Site.BaseURL == "" {
		return
	}

	domain := extractDomain(cfg.Site.BaseURL)
	if domain == "" {
		return
	}

	// Health: serviceName from site domain
	if cfg.Health.ServiceName == "" {
		cfg.Health.ServiceName = strings.ReplaceAll(domain, ".", "-")
	}

	// Analytics: plausible domain defaults to site domain
	if cfg.Analytics.Plausible != nil && cfg.Analytics.Plausible.Domain == "" {
		cfg.Analytics.Plausible.Domain = domain
	}

	// Mail: fromName defaults to site.name
	if cfg.Mail.FromName == "" && cfg.Site.Name != "" {
		cfg.Mail.FromName = cfg.Site.Name
	}

	// Mail: from address defaults to noreply@domain (override in ops config)
	if cfg.Mail.From == "" {
		cfg.Mail.From = "noreply@" + domain
	}
}

// extractDomain returns the hostname from a URL string, stripping port if present.
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname() // strips port
	if host == "" || host == "localhost" {
		return ""
	}
	return host
}

// mergePages overlays pages YAML data onto an existing SiteConfig.
// Fields from the pages file overwrite zero-value fields in cfg.
func mergePages(cfg *SiteConfig, data []byte) error {
	expanded := expandEnvSafe(string(data))

	overlay := &SiteConfig{}
	if err := yaml.Unmarshal([]byte(expanded), overlay); err != nil {
		return fmt.Errorf("failed to parse pages YAML: %w", err)
	}

	// Site identity — pages file is authoritative
	if overlay.Site.Name != "" {
		cfg.Site.Name = overlay.Site.Name
	}
	if overlay.Site.BaseURL != "" {
		cfg.Site.BaseURL = overlay.Site.BaseURL
	}
	if overlay.Site.LogoPath != "" {
		cfg.Site.LogoPath = overlay.Site.LogoPath
	}
	if overlay.Site.FaviconPath != "" {
		cfg.Site.FaviconPath = overlay.Site.FaviconPath
	}
	if overlay.Site.CopyrightStartYear != 0 {
		cfg.Site.CopyrightStartYear = overlay.Site.CopyrightStartYear
	}
	cfg.Site.ShowLangFlags = overlay.Site.ShowLangFlags

	// I18n
	if overlay.I18n.DefaultLanguage != "" {
		cfg.I18n.DefaultLanguage = overlay.I18n.DefaultLanguage
	}
	if len(overlay.I18n.Languages) > 0 {
		cfg.I18n.Languages = overlay.I18n.Languages
	}

	// Content directory
	if overlay.ContentDir != "" {
		cfg.ContentDir = overlay.ContentDir
	}

	// Navigation
	if len(overlay.Navigation.Header) > 0 {
		cfg.Navigation.Header = overlay.Navigation.Header
	}
	if len(overlay.Navigation.Footer) > 0 {
		cfg.Navigation.Footer = overlay.Navigation.Footer
	}

	// UI strings
	if len(overlay.UI) > 0 {
		cfg.UI = overlay.UI
	}

	// Contact — pages file owns enabled state (behavior/design).
	// Ops config owns operational params (maxLinks, rateLimit).
	if overlay.Contact.Enabled {
		cfg.Contact.Enabled = true
	}
	if len(overlay.Contact.Recipients) > 0 {
		cfg.Contact.Recipients = overlay.Contact.Recipients
	}
	if overlay.Contact.RecipientName != "" {
		cfg.Contact.RecipientName = overlay.Contact.RecipientName
	}
	if len(overlay.Contact.Subject) > 0 {
		cfg.Contact.Subject = overlay.Contact.Subject
	}

	// Pages (inline, if any)
	if len(overlay.Pages) > 0 {
		cfg.Pages = overlay.Pages
	}

	// Consent i18n texts — pages file is authoritative for user-facing copy.
	// The operational config (config.yaml) owns enabled/categories; pages.yaml owns i18n.
	// If pages.yaml defines consent fully (enabled + categories), it takes precedence
	// so consumer repos can own consent config without duplicating it in ops config.
	if overlay.Consent.Enabled {
		cfg.Consent.Enabled = true
		if len(overlay.Consent.Categories) > 0 {
			cfg.Consent.Categories = overlay.Consent.Categories
		}
	}
	if len(overlay.Consent.I18n) > 0 {
		if cfg.Consent.I18n == nil {
			cfg.Consent.I18n = make(map[string]ConsentI18nConfig)
		}
		for lang, texts := range overlay.Consent.I18n {
			cfg.Consent.I18n[lang] = texts
		}
	}

	// Analytics — pages file can define analytics config so consumer repos
	// own their analytics setup without needing it in the ops config.
	if overlay.Analytics.Plausible != nil {
		cfg.Analytics.Plausible = overlay.Analytics.Plausible
	}
	if overlay.Analytics.Collector != nil {
		cfg.Analytics.Collector = overlay.Analytics.Collector
	}

	// SEO — pages file can define site-wide SEO defaults.
	if overlay.SEO.DefaultOGImage != "" {
		cfg.SEO.DefaultOGImage = overlay.SEO.DefaultOGImage
	}
	if overlay.SEO.DefaultTwitterCard != "" {
		cfg.SEO.DefaultTwitterCard = overlay.SEO.DefaultTwitterCard
	}
	if len(overlay.SEO.GlobalJSONLD) > 0 {
		cfg.SEO.GlobalJSONLD = overlay.SEO.GlobalJSONLD
	}

	// Mail identity — from/fromName are site identity, not credentials.
	// Pages file can define these so ops config only needs host/port/credentials.
	if overlay.Mail.From != "" {
		cfg.Mail.From = overlay.Mail.From
	}
	if overlay.Mail.FromName != "" {
		cfg.Mail.FromName = overlay.Mail.FromName
	}
	if len(overlay.Mail.Templates) > 0 {
		if cfg.Mail.Templates == nil {
			cfg.Mail.Templates = make(map[string]MailTemplateConfig)
		}
		for lang, tmpl := range overlay.Mail.Templates {
			cfg.Mail.Templates[lang] = tmpl
		}
	}

	return nil
}

// Parse parses a YAML byte slice into a SiteConfig.
// Applies defaults and validates.
func Parse(data []byte) (*SiteConfig, error) {
	cfg, err := parseRaw(data)
	if err != nil {
		return nil, err
	}

	applyDefaults(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// parseRaw parses YAML and applies defaults but skips validation and default merging.
// Used internally by Load() which handles pagesDir loading before validation.
func parseRaw(data []byte) (*SiteConfig, error) {
	expanded := expandEnvSafe(string(data))

	cfg := &SiteConfig{}
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	return cfg, nil
}

// expandEnvSafe expands ${VAR} and $VAR but handles ${VAR:default} syntax
// that os.ExpandEnv does not support.
func expandEnvSafe(s string) string {
	return os.Expand(s, func(key string) string {
		// Handle ${VAR:default} syntax
		if idx := strings.Index(key, ":"); idx >= 0 {
			envKey := key[:idx]
			defaultVal := key[idx+1:]
			if v := os.Getenv(envKey); v != "" {
				return v
			}
			return defaultVal
		}
		return os.Getenv(key)
	})
}

// validate checks the config for required fields and consistency.
func validate(cfg *SiteConfig) error {
	if cfg.Site.Name == "" {
		return fmt.Errorf("site.name is required")
	}
	if cfg.Site.BaseURL == "" {
		return fmt.Errorf("site.baseURL is required")
	}

	// Pages are loaded separately via content.Load() — validate if present.
	// Pages without an i18n block are treated as supplementary entries (e.g. providing
	// SEO JSON-LD overrides) and are merged with content pages at startup; skip them here.
	if len(cfg.Pages) > 0 {
		// Validate i18n consistency — only for pages that declare i18n data.
		for _, lang := range cfg.I18n.Languages {
			for _, page := range cfg.Pages {
				if len(page.I18n) == 0 {
					// Supplementary page (SEO/JSON-LD only) — will be merged with content later.
					continue
				}
				if _, ok := page.I18n[lang]; !ok {
					return fmt.Errorf("page %q is missing i18n config for language %q", page.ID, lang)
				}
				// Empty slug is allowed only for the home page (single-page mode).
				if page.I18n[lang].Slug == "" && page.ID != "home" {
					return fmt.Errorf("page %q has empty slug for language %q (only id \"home\" may use an empty slug for single-page mode)", page.ID, lang)
				}
			}
		}

		// Validate no duplicate slugs within the same language.
		// Empty slug (single-page mode) is unique by definition — skip it.
		for _, lang := range cfg.I18n.Languages {
			slugs := make(map[string]string) // slug → page ID
			for _, page := range cfg.Pages {
				slug := page.I18n[lang].Slug
				if slug == "" {
					continue // root page — cannot conflict
				}
				if existing, ok := slugs[slug]; ok {
					return fmt.Errorf("duplicate slug %q for language %q: pages %q and %q", slug, lang, existing, page.ID)
				}
				slugs[slug] = page.ID
			}
		}
	}

	// Gate validation
	if cfg.Gate.Enabled {
		if cfg.Gate.CookieSecret == "" {
			return fmt.Errorf("gate.cookieSecret is required when gate.enabled=true")
		}
		if len(cfg.Gate.Codes) == 0 {
			return fmt.Errorf("gate.codes must contain at least one entry when gate.enabled=true")
		}
	}

	// ArconGate validation
	if cfg.ArconGate.Enabled {
		if cfg.ArconGate.CookieSecret == "" {
			return fmt.Errorf("arconGate.cookieSecret is required when arconGate.enabled=true")
		}
		if len(cfg.ArconGate.Codes) == 0 {
			return fmt.Errorf("arconGate.codes must contain at least one entry when arconGate.enabled=true")
		}
		if cfg.ArconGate.ContentDir == "" {
			return fmt.Errorf("arconGate.contentDir is required when arconGate.enabled=true")
		}
	}

	return nil
}
