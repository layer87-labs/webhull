package config

import (
	"fmt"
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

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
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

	// Contact (recipients and subjects are content, not operational)
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

	return nil
}

// Parse parses a YAML byte slice into a SiteConfig.
// Applies defaults and validates.
func Parse(data []byte) (*SiteConfig, error) {
	cfg, err := parseRaw(data)
	if err != nil {
		return nil, err
	}

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

	applyDefaults(cfg)
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

// applyDefaults sets sensible defaults for missing values.
func applyDefaults(cfg *SiteConfig) {
	if cfg.Server.Port == "" {
		cfg.Server.Port = envOrDefault("PORT", "8080")
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = envOrDefault("HOST", "0.0.0.0")
	}
	if cfg.Server.Environment == "" {
		cfg.Server.Environment = envOrDefault("ENVIRONMENT", "development")
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
		cfg.Server.StaticCacheMaxAge = 365 * 24 * time.Hour
	}

	if cfg.I18n.DefaultLanguage == "" {
		cfg.I18n.DefaultLanguage = "de"
	}
	if len(cfg.I18n.Languages) == 0 {
		cfg.I18n.Languages = []string{"de", "en"}
	}

	if cfg.Site.CopyrightStartYear == 0 {
		cfg.Site.CopyrightStartYear = time.Now().Year()
	}

	if cfg.Contact.MaxLinks == 0 {
		cfg.Contact.MaxLinks = 2
	}
	if cfg.Contact.RateLimit.Requests == 0 {
		cfg.Contact.RateLimit.Requests = 3
	}
	if cfg.Contact.RateLimit.Window == 0 {
		cfg.Contact.RateLimit.Window = 15 * time.Minute
	}

	// SMTP defaults
	if cfg.Mail.Port == 0 {
		cfg.Mail.Port = 587
	}

	// Gate defaults
	if cfg.Gate.CookieName == "" {
		cfg.Gate.CookieName = "gate_session"
	}
	if cfg.Gate.CookieMaxAge == 0 {
		cfg.Gate.CookieMaxAge = 24 * time.Hour
	}

	// ArconGate defaults
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
}

// validate checks the config for required fields and consistency.
func validate(cfg *SiteConfig) error {
	if cfg.Site.Name == "" {
		return fmt.Errorf("site.name is required")
	}
	if cfg.Site.BaseURL == "" {
		return fmt.Errorf("site.baseURL is required")
	}

	// Pages are loaded separately via content.Load() — validate if present
	if len(cfg.Pages) > 0 {
		// Validate i18n consistency
		for _, lang := range cfg.I18n.Languages {
			for _, page := range cfg.Pages {
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

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
