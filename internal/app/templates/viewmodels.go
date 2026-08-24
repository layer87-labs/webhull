package templates

import (
	"regexp"
	"strings"

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

	// PluginContent holds content-key → rendered-HTML fragments injected by
	// the plugin system for this page, read fresh per request from the
	// plugin service's cache. Takes precedence over Page.Content so a
	// plugin can supply a key without mutating the shared, concurrently-read
	// *pages.Page — see Content()/HasContent() below.
	PluginContent map[string]string
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

	// State is the visitor's current consent state. It is used to pre-check the
	// category toggles when the banner is reopened to change an earlier
	// decision. Nil is treated as "nothing decided yet".
	State *consent.State
}

// IsChecked reports whether the toggle for the given category should render
// checked. Required categories are always on; once the visitor has decided,
// their stored choice wins over the configured default.
func (cb *ConsentBannerData) IsChecked(cat consent.Category) bool {
	if cat.Required {
		return true
	}
	if cb.State != nil && cb.State.Decided {
		return cb.State.Categories[cat.Key]
	}
	return cat.Default
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

// pluginMarkerPattern matches an inline plugin injection point, e.g.
// "<!-- plugin: fleet -->", written directly inside a raw HTML content
// body. Mirrors the existing "<!-- section: name -->" marker syntax.
var pluginMarkerPattern = regexp.MustCompile(`<!--\s*plugin:\s*([a-zA-Z0-9_-]+)\s*-->`)

// Content returns a content value by key, or empty string if not found.
//
// A plugin can supply content two ways:
//   - As the entire value for a content key (e.g. a page built from typed
//     or named sections calling Content("fleet") directly) — takes
//     precedence over the page's own frontmatter content for that key.
//   - As an inline "<!-- plugin: fleet -->" marker inside any content
//     value, including the raw-body-fallback "body" key used by
//     fully-custom pages that manage their own HTML structure. Every
//     returned value is scanned for markers and substituted.
func (pd *PageData) Content(key string) string {
	if v, ok := pd.PluginContent[key]; ok {
		return v
	}
	var raw string
	if pd.Page != nil && pd.Page.Content != nil {
		raw = pd.Page.Content[key]
	}
	if raw == "" || !strings.Contains(raw, "<!--") {
		return raw
	}
	return pluginMarkerPattern.ReplaceAllStringFunc(raw, func(m string) string {
		sub := pluginMarkerPattern.FindStringSubmatch(m)
		return pd.PluginContent[sub[1]]
	})
}

// HasContent checks if a content key exists and has a non-empty value,
// checking plugin-supplied content first (see Content).
func (pd *PageData) HasContent(key string) bool {
	if v, ok := pd.PluginContent[key]; ok {
		return v != ""
	}
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
	if cfg.SubmitIcon != "" {
		defaults.SubmitIcon = cfg.SubmitIcon
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

// RenderConsentBanner returns true if the consent banner markup should be
// emitted at all.
//
// This is deliberately broader than ShowConsentBanner: the markup is also
// emitted once the visitor has decided, so the decision can be reopened and
// withdrawn at any time (GDPR Art. 7(3), TTDSG §25). In that case the banner
// starts hidden and is revealed by any [data-consent-open] trigger.
func (pd *PageData) RenderConsentBanner() bool {
	return pd.ConsentConfig != nil &&
		pd.ConsentConfig.Enabled &&
		pd.Consent != nil &&
		!pd.Consent.Bypassed
}

// ConsentDecided returns true when the visitor has already made a choice, in
// which case the banner renders hidden rather than open.
func (pd *PageData) ConsentDecided() bool {
	return pd.Consent != nil && pd.Consent.Decided
}

// ClientAnalyticsAllowed reports whether client-side analytics scripts may be
// injected into the page.
//
// Injecting them is itself an act of tracking: the Plausible script sends a
// pageview the moment it loads. It therefore requires an explicit accepted
// analytics consent, not merely a configured provider. Without consent the
// server falls back to anonymous server-side pageview tracking — see
// analytics.Service.TrackServerSide.
func (pd *PageData) ClientAnalyticsAllowed() bool {
	// Consent management switched off site-wide: there is nothing to gate on,
	// and the operator has made that call deliberately.
	if pd.ConsentConfig == nil || !pd.ConsentConfig.Enabled {
		return true
	}
	return pd.Consent != nil && pd.Consent.IsAllowed(consent.CategoryAnalytics)
}

// ConsentSettingsLabel returns the label for the footer link that reopens the
// consent dialog, or an empty string when no link should be rendered.
//
// It falls back to the consent banner title so that a site which enables
// consent always gets a working withdrawal path, even without extra config.
func (pd *PageData) ConsentSettingsLabel() string {
	if !pd.RenderConsentBanner() {
		return ""
	}
	if pd.UI.ConsentSettingsLabel != "" {
		return pd.UI.ConsentSettingsLabel
	}
	return pd.ConsentConfig.Texts.Title
}
