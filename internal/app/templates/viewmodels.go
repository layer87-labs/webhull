package templates

import (
	"github.com/layer87-labs/webhull/internal/pkg/assets"
	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/consent"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
	"github.com/layer87-labs/webhull/internal/pkg/navigation"
	"github.com/layer87-labs/webhull/internal/pkg/pages"
	"github.com/layer87-labs/webhull/internal/pkg/seo"
)

// PageData holds all data required to render a complete page.
// This is the single view model passed from handler to templ layout.
type PageData struct {
	// Page is the resolved page from the pages service.
	Page *pages.Page

	// Meta holds all SEO meta tags (OG, Twitter, hreflang, canonical).
	Meta seo.MetaTags

	// Header is the resolved header navigation with active state.
	Header navigation.Header

	// Footer is the resolved footer navigation with active state.
	Footer navigation.Footer

	// Copyright is the formatted copyright year string.
	Copyright string

	// LangLinks holds language switcher entries.
	LangLinks []seo.LanguageSwitchLink

	// Consent holds the current consent state.
	Consent *consent.State

	// ConsentConfig holds the consent banner configuration for rendering.
	ConsentConfig *ConsentBannerData

	// Site holds global site identity (name, logo, favicon).
	Site SiteData

	// Analytics holds analytics configuration for script injection.
	Analytics AnalyticsData

	// UI holds global per-language UI strings (contact CTA, footer, etc.).
	UI config.UIStringsConfig

	// IsBot indicates if the current request is from a known bot.
	IsBot bool

	// ContactEnabled indicates whether the contact form is active.
	// Used by the single-page template to conditionally render the contact section.
	ContactEnabled bool

	// Assets provides cache-busted asset paths.
	Assets *assets.Service
}

// SiteData holds global site identity for template rendering.
type SiteData struct {
	Name          string
	BaseURL       string
	LogoPath      string
	FaviconPath   string
	ShowLangFlags bool
}

// AnalyticsData holds analytics config for conditional script injection.
type AnalyticsData struct {
	PlausibleEnabled bool
	PlausibleDomain  string
	PlausibleScript  string
	CollectorEnabled bool
	CollectorScript  string
}

// GatePageData holds the view model for the access gate page.
// This is intentionally minimal — the gate page has no header, footer, or nav.
type GatePageData struct {
	// SiteName is displayed in the page title.
	SiteName string

	// LogoPath is the URL path to the site logo image.
	LogoPath string

	// Error indicates whether the previous code submission failed.
	Error bool

	// ErrorMsg is the human-readable error message shown to the user.
	ErrorMsg string

	// Redirect is the URL the user originally requested before being gated.
	// It is passed through the form as a hidden field and used for post-login redirect.
	Redirect string

	// FormAction is the POST URL for the login form.
	// Defaults to "/gate" when empty. Use "/arcon/gate" for the arcon-specific gate.
	FormAction string
}

// ConsentBannerData holds all data for the consent banner component.
type ConsentBannerData struct {
	Enabled    bool
	Texts      config.ConsentI18nConfig
	Categories []consent.Category
}

// Language returns the current page language.
func (pd *PageData) Language() i18n.Language {
	if pd.Page != nil {
		return pd.Page.Language
	}
	return i18n.LangDE
}

// LangCode returns the current language as string for HTML lang attribute.
func (pd *PageData) LangCode() string {
	return pd.Language().String()
}

// Content returns a content value by key, or empty string if not found.
func (pd *PageData) Content(key string) string {
	if pd.Page != nil && pd.Page.Content != nil {
		return pd.Page.Content[key]
	}
	return ""
}

// HasContent checks if a content key exists and has a non-empty value.
func (pd *PageData) HasContent(key string) bool {
	if pd.Page != nil && pd.Page.Content != nil {
		v, ok := pd.Page.Content[key]
		return ok && v != ""
	}
	return false
}

// AssetPath returns a cache-busted asset URL path.
// e.g. "/static/css/style.css" → "/static/css/style.css?v=a1b2c3d4"
func (pd *PageData) AssetPath(path string) string {
	if pd.Assets != nil {
		return pd.Assets.Path(path)
	}
	return path
}

// ResolveContactTexts returns localised contact form texts for the current page language.
// Non-empty values from ui.contactForm in site config override the built-in defaults.
func (pd *PageData) ResolveContactTexts() ContactFormTexts {
	defaults := DefaultContactTexts(pd.LangCode())
	cfg := pd.UI.ContactForm
	if cfg.Heading != "" {
		defaults.Heading = cfg.Heading
	}
	if cfg.SubmitText != "" {
		defaults.SubmitText = cfg.SubmitText
	}
	if cfg.SuccessMsg != "" {
		defaults.SuccessMsg = cfg.SuccessMsg
	}
	if cfg.SuccessRefMsg != "" {
		defaults.SuccessRefMsg = cfg.SuccessRefMsg
	}
	if cfg.ErrorMsg != "" {
		defaults.ErrorMsg = cfg.ErrorMsg
	}
	if len(cfg.Fields) > 0 {
		defaults.Fields = cfg.Fields
	}
	return defaults
}

// ShowConsentBanner returns true if the consent banner should be displayed.
// Returns false when the request was handled by the server-side bypass
// middleware (automated tools, Lighthouse, Unlighthouse, Playwright).
func (pd *PageData) ShowConsentBanner() bool {
	return pd.ConsentConfig != nil &&
		pd.ConsentConfig.Enabled &&
		pd.Consent != nil &&
		!pd.Consent.Decided &&
		!pd.Consent.Bypassed
}

// ConsentBypassed returns true when the current request was handled by the
// server-side consent bypass middleware. Used by the layout template to
// suppress consent.js inclusion for automated audit tools.
func (pd *PageData) ConsentBypassed() bool {
	return pd.Consent != nil && pd.Consent.Bypassed
}
