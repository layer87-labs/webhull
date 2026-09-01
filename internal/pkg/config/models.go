package config

import "time"

// SiteConfig is the root configuration loaded from YAML.
// It defines the complete website: pages, languages, navigation, features.
type SiteConfig struct {
	// Site identity
	Site SiteIdentity `yaml:"site"`

	// Internationalization
	I18n I18nConfig `yaml:"i18n"`

	// Navigation (header + footer)
	Navigation NavigationConfig `yaml:"navigation"`

	// Global per-language UI strings (contact CTA, footer links, etc.)
	UI map[string]UIStringsConfig `yaml:"ui"`

	// Pages loaded from content directory (populated at runtime by content.Load)
	Pages []PageConfig `yaml:"pages,omitempty"`

	// ContentDir is the directory containing HTML content files with YAML frontmatter.
	// Relative to the config file's directory. Structure: contentDir/{lang}/*.html
	ContentDir string `yaml:"contentDir,omitempty"`

	// PluginsDir is the directory containing plugin subdirectories, each with
	// its own plugin.yaml manifest. Relative to the config file's directory
	// (same resolution rule as ContentDir). Empty disables the plugin system.
	PluginsDir string `yaml:"pluginsDir,omitempty"`

	// Contact form
	Contact ContactConfig `yaml:"contact"`

	// Analytics providers
	Analytics AnalyticsConfig `yaml:"analytics"`

	// Observability — error tracking in the browser and CSP violation
	// reporting. Both targets are configuration, never compiled in: a fixed
	// endpoint binds the binary to one installation and turns it into a tool
	// that reports into someone else's system when run elsewhere.
	Observability ObservabilityConfig `yaml:"observability"`

	// Consent / Cookie management
	Consent ConsentConfig `yaml:"consent"`

	// SEO defaults
	SEO SEODefaults `yaml:"seo"`

	// Server settings
	Server ServerConfig `yaml:"server"`

	// Health server settings
	Health HealthConfig `yaml:"health"`

	// Mail settings
	Mail MailConfig `yaml:"mail"`

	// Access Gate (optional) — protects the entire site behind a code-based login.
	// When disabled (default), the site is fully public.
	//
	// Example site.yaml configuration:
	//
	//   gate:
	//     enabled: true
	//     cookieSecret: "${GATE_SECRET}"
	//     cookieName: "gate_session"
	//     cookieMaxAge: 24h
	//     codes:
	//       - code: "aurora-pine-7"
	//         label: "Interessent Mustermann AG"
	//       - code: "silver-lake-3"
	//         label: "Partner XYZ GmbH"
	Gate GateConfig `yaml:"gate"`

	// ArconGate (optional) — protects the /arcon/* path behind a code-based login.
	// Independent of the global Gate: the rest of the site stays public.
	// ContentDir is the directory (relative to the config file) whose files are
	// served under /arcon/. Omitting contentDir disables file serving even when
	// enabled=true.
	//
	// Example site.yaml configuration:
	//
	//   arconGate:
	//     enabled: true
	//     contentDir: "arcon"
	//     cookieSecret: "${ARCON_GATE_SECRET}"
	//     cookieName: "arcon_session"
	//     cookieMaxAge: 8h
	//     title: "ARCON – Layer87"
	//     codes:
	//       - code: "cedar-frost-9"
	//         label: "Demo Interessent"
	ArconGate ArconGateConfig `yaml:"arconGate"`
}

// SiteIdentity defines the website's identity.
type SiteIdentity struct {
	Name               string `yaml:"name"`
	BaseURL            string `yaml:"baseURL"`
	LogoPath           string `yaml:"logoPath"`
	FaviconPath        string `yaml:"faviconPath"`
	CopyrightStartYear int    `yaml:"copyrightStartYear"`
	ShowLangFlags      bool   `yaml:"showLangFlags"`
}

// I18nConfig defines available languages and defaults.
type I18nConfig struct {
	DefaultLanguage string   `yaml:"defaultLanguage"`
	Languages       []string `yaml:"languages"`
}

// NavigationConfig defines header and footer navigation per language.
type NavigationConfig struct {
	Header map[string][]NavItemConfig `yaml:"header"` // key = language code
	Footer map[string][]NavItemConfig `yaml:"footer"` // key = language code
}

// NavItemConfig is a single navigation entry.
type NavItemConfig struct {
	Slug     string          `yaml:"slug"`
	Title    string          `yaml:"title"`
	URL      string          `yaml:"url"`
	Children []NavItemConfig `yaml:"children,omitempty"`
}

// UIStringsConfig holds global per-language strings for header, footer, and shared UI.
type UIStringsConfig struct {
	ContactURL    string `yaml:"contactURL"`
	ContactLabel  string `yaml:"contactLabel"`
	FooterTagline string `yaml:"footerTagline"`
	AllRights     string `yaml:"allRights"`
	ImprintURL    string `yaml:"imprintURL"`
	ImprintLabel  string `yaml:"imprintLabel"`
	PrivacyURL    string `yaml:"privacyURL"`
	PrivacyLabel  string `yaml:"privacyLabel"`
	// CreditsText is an optional attribution line shown at the end of the footer bar.
	// Example: "Designed & hosted by Layer87"
	CreditsText string `yaml:"creditsText,omitempty"`
	// CreditsURL is an optional URL that wraps CreditsText in a link.
	CreditsURL string `yaml:"creditsURL,omitempty"`
	// ConsentSettingsLabel overrides the label of the footer link that reopens
	// the cookie consent dialog. When empty, the consent banner title is used.
	// The link is rendered whenever consent is enabled, so that an earlier
	// decision can always be withdrawn (GDPR Art. 7(3)).
	ConsentSettingsLabel string `yaml:"consentSettingsLabel,omitempty"`
	NotFoundTitle        string `yaml:"notFoundTitle"`
	NotFoundSubtitle     string `yaml:"notFoundSubtitle"`
	NotFoundButton       string `yaml:"notFoundButton"`
	// ContactForm holds optional per-language overrides for contact form labels and messages.
	// Any field left empty falls back to the built-in DE/EN defaults.
	ContactForm ContactFormConfig `yaml:"contactForm,omitempty"`
}

// ContactFormConfig holds optional localised overrides for the contact form UI.
// Only non-empty values override the built-in defaults — it is not necessary
// to set every field. Useful for non-DE/EN languages or custom copy.
// When Fields is non-empty it replaces ALL default fields — the site fully
// controls which inputs appear and in what order.
type ContactFormConfig struct {
	Heading       string `yaml:"heading,omitempty"`
	SubmitText    string `yaml:"submitText,omitempty"`
	SubmitIcon    string `yaml:"submitIcon,omitempty"`
	SuccessMsg    string `yaml:"successMsg,omitempty"`
	SuccessRefMsg string `yaml:"successRefMsg,omitempty"`
	ErrorMsg      string `yaml:"errorMsg,omitempty"`
	// Fields replaces the default Name/Email/Subject/Message fields when non-empty.
	Fields []FieldConfig `yaml:"fields,omitempty"`
}

// FieldConfig defines a single input field rendered in the contact form.
type FieldConfig struct {
	// Name is the HTML input name attribute and the JSON key sent to the server.
	Name string `yaml:"name"`
	// Label is the display label shown above the input.
	Label string `yaml:"label"`
	// Type is the HTML input type or special type: "text", "email", "tel", "textarea", "select". Defaults to "text".
	Type string `yaml:"type,omitempty"`
	// Required marks the field as mandatory.
	Required bool `yaml:"required,omitempty"`
	// Placeholder is optional hint text shown inside the input.
	Placeholder string `yaml:"placeholder,omitempty"`
	// Options lists the selectable values for type "select".
	// Each entry is the option value (also used as display label).
	Options []string `yaml:"options,omitempty"`
}

// PageConfig defines a single page with all language variants.
// Can be defined inline in site.yaml or in a separate YAML file under pagesDir.
type PageConfig struct {
	// Unique internal ID for cross-referencing (e.g., "contact", "home")
	ID string `yaml:"id"`

	// Template name to use for rendering (e.g., "contact", "legal", "home")
	Template string `yaml:"template"`

	// Per-language configuration
	I18n map[string]PageI18nConfig `yaml:"i18n"`

	// SEO overrides (per-language is in PageI18nConfig)
	SEO PageSEOConfig `yaml:"seo,omitempty"`
}

// PageI18nConfig holds language-specific page data.
type PageI18nConfig struct {
	Slug        string `yaml:"slug"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Keywords    string `yaml:"keywords,omitempty"`
	// SEOTitle overrides the page <title> and og:title for this language.
	// Uses the same key as the HTML frontmatter field seo_title.
	SEOTitle string `yaml:"seo_title,omitempty"`
	// SEODesc overrides the meta description and og:description for this language.
	// Uses the same key as the HTML frontmatter field seo_description.
	SEODesc string            `yaml:"seo_description,omitempty"`
	Content map[string]string `yaml:"content,omitempty"` // arbitrary key-value for template interpolation
	// Sections holds ordered typed content sections parsed from the body HTML.
	// Populated at runtime by content.Load when typed section markers are found.
	Sections []SectionConfig `yaml:"sections,omitempty"`
}

// SectionConfig represents a single typed content section in render order.
// Parsed from <!-- section[type,altbg,id=anchor]: Title HTML --> markers in the body.
type SectionConfig struct {
	// Type controls the CSS layout: "block", "grid", or "services".
	Type string `yaml:"type"`
	// AltBg applies the alt-bg CSS modifier to the outer section element.
	AltBg bool `yaml:"altbg,omitempty"`
	// ID is an optional HTML id attribute placed on the outer section element.
	ID string `yaml:"id,omitempty"`
	// Title is raw HTML rendered in the section header <h2>. Empty means no header.
	Title string `yaml:"title"`
	// Body is raw HTML rendered as the section content.
	Body string `yaml:"body"`
}

// PageSEOConfig holds SEO settings per page.
type PageSEOConfig struct {
	Priority   float64 `yaml:"priority"`
	ChangeFreq string  `yaml:"changefreq"`
	OGImage    string  `yaml:"ogImage,omitempty"`
	OGType     string  `yaml:"ogType,omitempty"`
	NoIndex    bool    `yaml:"noIndex,omitempty"`
	// JSONLD holds zero or more raw JSON-LD blocks injected as
	// <script type="application/ld+json"> in the page <head>.
	// These are merged with any GlobalJSONLD blocks from SEODefaults.
	JSONLD []string `yaml:"jsonLD,omitempty"`
}

// ContactConfig defines the contact form behavior.
type ContactConfig struct {
	Enabled       bool              `yaml:"enabled"`
	RecipientName string            `yaml:"recipientName"`
	Recipients    []string          `yaml:"recipients"` // email addresses
	Subject       map[string]string `yaml:"subject"`    // per-language subject template
	MaxLinks      int               `yaml:"maxLinks"`
	RateLimit     RateLimitConfig   `yaml:"rateLimit"`
}

// RateLimitConfig defines rate limiting parameters.
type RateLimitConfig struct {
	Requests int           `yaml:"requests"`
	Window   time.Duration `yaml:"window"`
}

// AnalyticsConfig supports multiple providers in parallel.
type AnalyticsConfig struct {
	Plausible *PlausibleConfig `yaml:"plausible,omitempty"`
	Collector *CollectorConfig `yaml:"collector,omitempty"`
}

// ObservabilityConfig wires the site into error tracking and CSP reporting.
//
// The two are separate signals that happen to share a backend. Error tracking
// closes the gap that server-side tracing leaves open by construction: a
// JavaScript exception that never reaches the server appears in no trace. CSP
// reporting is not an error channel at all — a report means the browser
// refused something, which is either an attack, a browser extension, or a
// rule that is too tight.
type ObservabilityConfig struct {
	ErrorTracking *ErrorTrackingConfig `yaml:"errorTracking,omitempty"`
	CSP           CSPConfig            `yaml:"csp"`
}

// ErrorTrackingConfig configures a Sentry-compatible browser SDK
// (Sentry, GlitchTip, ...).
type ErrorTrackingConfig struct {
	Enabled bool `yaml:"enabled"`

	// DSN is the Sentry-format ingest URL. It is public by design — it ends
	// up in every delivered page and is good for writing and nothing else.
	DSN string `yaml:"dsn"`

	// SDKURL is where the browser loads the SDK from. No default on purpose:
	// hardcoding a CDN would decide for every operator where their visitors'
	// browsers fetch code from. Serve it from the site's own static directory
	// (e.g. /assets/sentry.min.js) or name a CDN deliberately.
	SDKURL string `yaml:"sdkURL"`

	// Environment separates production from staging in the backend. Falls back
	// to server.environment when empty.
	Environment string `yaml:"environment"`

	// Release tags events with a version, so a spike can be tied to a deploy.
	Release string `yaml:"release"`

	// SampleRate between 0 and 1. Errors are rare; the default of 1 is the
	// right one for almost every site.
	SampleRate float64 `yaml:"sampleRate"`
}

// CSPConfig configures Content-Security-Policy reporting and enforcement.
type CSPConfig struct {
	// ReportURI receives violation reports. A relative path is the right
	// choice: it resolves same-origin, which is preflight-free today with
	// report-uri and stays preflight-free after the move to
	// Reporting-Endpoints, where a cross-origin endpoint needs CORS.
	ReportURI string `yaml:"reportURI"`

	// ReportOnly sends the policy as Content-Security-Policy-Report-Only.
	// The browser then checks the rule, lets everything through, and reports
	// anyway — the only way to learn what a policy costs before enforcing it.
	ReportOnly bool `yaml:"reportOnly"`
}

// PlausibleConfig for Plausible Analytics proxy.
type PlausibleConfig struct {
	Enabled    bool   `yaml:"enabled"`
	BaseURL    string `yaml:"baseURL"`
	ScriptPath string `yaml:"scriptPath"`
	Domain     string `yaml:"domain"`
}

// CollectorConfig for the custom analytics collector endpoint.
type CollectorConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"` // backend URL to forward events to
}

// ConsentConfig defines cookie consent behavior.
type ConsentConfig struct {
	Enabled    bool                         `yaml:"enabled"`
	Categories map[string]ConsentCategory   `yaml:"categories"`
	I18n       map[string]ConsentI18nConfig `yaml:"i18n"` // per-language texts
}

// ConsentCategory defines a single cookie category.
type ConsentCategory struct {
	Required bool `yaml:"required"`
	Default  bool `yaml:"default"`
}

// ConsentI18nConfig holds consent banner texts per language.
type ConsentI18nConfig struct {
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	AcceptAll   string            `yaml:"acceptAll"`
	RejectAll   string            `yaml:"rejectAll"`
	Customize   string            `yaml:"customize"`
	Save        string            `yaml:"save"`
	Categories  map[string]string `yaml:"categories"` // category key → display name
}

// SEODefaults are site-wide SEO fallbacks.
type SEODefaults struct {
	DefaultOGImage     string `yaml:"defaultOGImage"`
	DefaultTwitterCard string `yaml:"defaultTwitterCard"`
	// GlobalJSONLD holds raw JSON-LD blocks injected on every page.
	// Use this for site-wide schemas like Organization and WebSite.
	GlobalJSONLD []string `yaml:"globalJSONLD,omitempty"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port              string        `yaml:"port"`
	Host              string        `yaml:"host"`
	Environment       string        `yaml:"environment"` // "development" or "production"
	ReadTimeout       time.Duration `yaml:"readTimeout"`
	WriteTimeout      time.Duration `yaml:"writeTimeout"`
	IdleTimeout       time.Duration `yaml:"idleTimeout"`
	ShutdownTimeout   time.Duration `yaml:"shutdownTimeout"`
	StaticDir         string        `yaml:"staticDir"`
	CacheMaxAge       time.Duration `yaml:"cacheMaxAge"`
	StaticCacheMaxAge time.Duration `yaml:"staticCacheMaxAge"`

	// HSTS extends the Strict-Transport-Security header sent in production.
	HSTS HSTSConfig `yaml:"hsts"`
}

// HSTSConfig extends Strict-Transport-Security beyond the plain max-age.
//
// Both fields default to false, and that default is the point.
// includeSubDomains applies to EVERY subdomain of the site, including future
// ones and ones hosted elsewhere; preload enters the domain into a list
// shipped inside browsers, where removal takes months. Neither belongs in a
// default, and neither should be switched on without knowing every hostname
// under the domain.
type HSTSConfig struct {
	IncludeSubdomains bool `yaml:"includeSubdomains"`
	Preload           bool `yaml:"preload"`
}

// HealthConfig holds health server settings.
type HealthConfig struct {
	Enabled        *bool         `yaml:"enabled"`
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	HealthPath     string        `yaml:"healthPath"`
	ReadyPath      string        `yaml:"readyPath"`
	MetricsPath    string        `yaml:"metricsPath"`
	Timeout        time.Duration `yaml:"timeout"`
	ServiceName    string        `yaml:"serviceName"`
	ServiceVersion string        `yaml:"serviceVersion"`
}

// MailConfig holds SMTP connection settings.
type MailConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	FromName string `yaml:"fromName"`
	UseTLS   bool   `yaml:"useTLS"`

	// Per-language mail templates for auto-reply
	Templates map[string]MailTemplateConfig `yaml:"templates"`
}

// GateConfig configures the optional access gate that protects the entire site.
// When enabled, every page request requires a valid signed session cookie.
// Codes are stored as plaintext in the configuration — no database required.
type GateConfig struct {
	// Enabled activates the access gate. Default: false.
	Enabled bool `yaml:"enabled"`

	// CookieName is the name of the session cookie. Default: "gate_session".
	CookieName string `yaml:"cookieName"`

	// CookieMaxAge is the session lifetime. Default: 24h.
	CookieMaxAge time.Duration `yaml:"cookieMaxAge"`

	// CookieSecret is the HMAC-SHA256 signing key. Required when enabled=true.
	// Use ${GATE_SECRET} to inject via environment variable.
	CookieSecret string `yaml:"cookieSecret"`

	// Codes is the list of valid plaintext access codes.
	// At least one entry is required when enabled=true.
	Codes []GateCode `yaml:"codes"`

	// MsgTooManyAttempts overrides the rate-limit error shown to the user.
	// Defaults to an English message when empty.
	MsgTooManyAttempts string `yaml:"msgTooManyAttempts,omitempty"`

	// MsgInvalidCode overrides the wrong-code error shown to the user.
	// Defaults to an English message when empty.
	MsgInvalidCode string `yaml:"msgInvalidCode,omitempty"`
}

// GateCode is a single plaintext access code with an internal descriptive label.
type GateCode struct {
	// Code is the plaintext access code entered by the user.
	Code string `yaml:"code"`

	// Label is an internal comment, e.g. "Interessent Mustermann AG".
	Label string `yaml:"label"`
}

// ArconGateConfig configures the optional ARCON experience gate that protects
// only the /arcon/* path. The rest of the site remains fully public.
type ArconGateConfig struct {
	// Enabled activates the arcon gate. Default: false.
	Enabled bool `yaml:"enabled"`

	// ContentDir is the directory containing the ARCON HTML experience files.
	// Resolved relative to the config file's directory.
	// Files are served at /arcon/<file>.
	ContentDir string `yaml:"contentDir"`

	// Title is shown in the gate page browser title. Default: "ARCON – Access".
	Title string `yaml:"title"`

	// CookieName is the name of the arcon session cookie. Default: "arcon_session".
	CookieName string `yaml:"cookieName"`

	// CookieMaxAge is the arcon session lifetime. Default: 8h.
	CookieMaxAge time.Duration `yaml:"cookieMaxAge"`

	// CookieSecret is the HMAC-SHA256 signing key. Required when enabled=true.
	// Use ${ARCON_GATE_SECRET} to inject via environment variable.
	CookieSecret string `yaml:"cookieSecret"`

	// Codes is the list of valid plaintext access codes.
	// At least one entry is required when enabled=true.
	Codes []GateCode `yaml:"codes"`
}

// MailTemplateConfig defines a response mail template per language.
type MailTemplateConfig struct {
	Subject  string `yaml:"subject"`
	Body     string `yaml:"body"`     // inline HTML body
	BodyFile string `yaml:"bodyFile"` // path to HTML file, relative to config dir; overrides Body if set
}
