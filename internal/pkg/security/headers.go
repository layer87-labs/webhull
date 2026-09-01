package security

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig configures the security headers middleware.
type SecurityHeadersConfig struct {
	// IsProduction enables HSTS and other production-only headers.
	IsProduction bool

	// PlausibleBaseURL is the Plausible Analytics base URL (e.g. "https://plausible.io").
	// If non-empty, the host is added to script-src and connect-src.
	PlausibleBaseURL string

	// ArconPathPrefix, when non-empty, applies a relaxed CSP for requests whose
	// path starts with this prefix (e.g. "/arcon"). The relaxed policy allows
	// inline scripts, inline styles, Google Fonts, and data URIs needed by the
	// ARCON experience pages.
	ArconPathPrefix string

	// ExtraImgSrc lists additional origins allowed in img-src, e.g. a
	// plugin's CDN host for externally-sourced images.
	ExtraImgSrc []string

	// CSPReportURI, when non-empty, appends `report-uri <value>` to every
	// policy. Without it the policy is enforced silently and nobody learns
	// what it blocks — which is the interesting half: every violation is
	// either an attack that was stopped or a page that is broken.
	CSPReportURI string

	// CSPReportOnly sends the policy as Content-Security-Policy-Report-Only
	// instead of enforcing it. Nothing is blocked, everything is reported.
	CSPReportOnly bool

	// ErrorTrackingOrigin is the scheme+host the browser SDK posts events to,
	// derived from the DSN. It has to be in connect-src or the SDK's own
	// requests are the first thing the policy blocks.
	ErrorTrackingOrigin string

	// ErrorTrackingSDKOrigin is the scheme+host the SDK itself is loaded
	// from, when that is not the site's own origin.
	ErrorTrackingSDKOrigin string

	// HSTSIncludeSubdomains and HSTSPreload extend the HSTS header. Both
	// default to false, and that default is deliberate: includeSubDomains
	// applies to every subdomain including future ones and ones hosted
	// elsewhere, and preload enters the domain into a list shipped inside
	// browsers, where removal takes months. Neither belongs in a default.
	HSTSIncludeSubdomains bool
	HSTSPreload           bool
}

// SecurityHeadersMiddleware sets common security headers including CSP.
func SecurityHeadersMiddleware(cfg SecurityHeadersConfig) gin.HandlerFunc {
	// Pre-build CSPs at init time (static per config).
	csp := buildCSP(cfg)
	arconCSP := buildArconCSP(cfg)

	cspHeader := "Content-Security-Policy"
	if cfg.CSPReportOnly {
		cspHeader = "Content-Security-Policy-Report-Only"
	}
	hsts := buildHSTS(cfg)

	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		if cfg.IsProduction {
			c.Header("Strict-Transport-Security", hsts)
		}

		// Use relaxed CSP for the ARCON experience path (inline scripts + Google Fonts).
		if cfg.ArconPathPrefix != "" && strings.HasPrefix(c.Request.URL.Path, cfg.ArconPathPrefix) {
			c.Header(cspHeader, arconCSP)
		} else {
			c.Header(cspHeader, csp)
		}

		c.Next()
	}
}

// buildCSP constructs the Content-Security-Policy header value.
func buildCSP(cfg SecurityHeadersConfig) string {
	scriptSrc := "'self'"
	connectSrc := "'self'"

	// If Plausible is configured, allow its domain for script loading and API calls.
	if cfg.PlausibleBaseURL != "" {
		if host := extractHost(cfg.PlausibleBaseURL); host != "" {
			scriptSrc += " " + host
			connectSrc += " " + host
		}
	}

	// The SDK posts events to the DSN host and is loaded from the SDK host.
	// Both have to be allowed or the error tracker is the first thing the
	// policy breaks — a failure mode that is invisible, because the report
	// about it cannot be sent either.
	if cfg.ErrorTrackingOrigin != "" {
		connectSrc += " " + cfg.ErrorTrackingOrigin
	}
	if cfg.ErrorTrackingSDKOrigin != "" {
		scriptSrc += " " + cfg.ErrorTrackingSDKOrigin
	}

	imgSrc := "'self' data:"
	for _, host := range cfg.ExtraImgSrc {
		imgSrc += " " + host
	}

	directives := []string{
		"default-src 'self'",
		"script-src " + scriptSrc,
		"style-src 'self' 'unsafe-inline'",
		"img-src " + imgSrc,
		"font-src 'self'",
		"connect-src " + connectSrc,
		"frame-src 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}
	directives = appendReportURI(directives, cfg.CSPReportURI)

	return strings.Join(directives, "; ")
}

// appendReportURI adds the report-uri directive when a target is configured.
//
// report-uri is deprecated in favour of the Reporting API, but it is the one
// mechanism every current browser still honours, and a relative path keeps
// the endpoint same-origin — which the successor requires CORS for when it
// is not.
func appendReportURI(directives []string, reportURI string) []string {
	if reportURI == "" {
		return directives
	}
	return append(directives, "report-uri "+reportURI)
}

// buildHSTS assembles the Strict-Transport-Security value.
func buildHSTS(cfg SecurityHeadersConfig) string {
	v := "max-age=31536000"
	if cfg.HSTSIncludeSubdomains {
		v += "; includeSubDomains"
	}
	if cfg.HSTSPreload {
		v += "; preload"
	}
	return v
}

// buildArconCSP constructs the relaxed CSP used for the ARCON experience pages.
// These pages use inline scripts (canvas animations), inline event handlers,
// and Google Fonts — none of which are compatible with a strict CSP.
func buildArconCSP(cfg SecurityHeadersConfig) string {
	scriptSrc := "'self' 'unsafe-inline'"
	connectSrc := "'self'"
	if cfg.ErrorTrackingOrigin != "" {
		connectSrc += " " + cfg.ErrorTrackingOrigin
	}
	if cfg.ErrorTrackingSDKOrigin != "" {
		scriptSrc += " " + cfg.ErrorTrackingSDKOrigin
	}

	directives := []string{
		"default-src 'self'",
		"script-src " + scriptSrc,
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data:",
		"connect-src " + connectSrc,
		"frame-src 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}
	directives = appendReportURI(directives, cfg.CSPReportURI)
	return strings.Join(directives, "; ")
}

// OriginOf returns the scheme://host of a URL, or "" for a relative path or
// an unparseable value. A relative SDK or DSN path is same-origin and needs
// no entry in the policy, so "" is the correct answer there and not a
// failure.
func OriginOf(rawURL string) string {
	return extractHost(rawURL)
}

// extractHost returns the scheme+host from a URL (e.g. "https://plausible.io").
func extractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// BotDetector detects known bots from User-Agent strings.
type BotDetector struct {
	patterns []string
}

// NewBotDetector creates a bot detector with default patterns.
func NewBotDetector() *BotDetector {
	return &BotDetector{
		patterns: []string{
			"googlebot", "bingbot", "baiduspider", "yandex",
			"ahrefs", "semrush", "crawler", "spider", "bot",
			"mj12bot", "dotbot", "petalbot", "serpstatbot",
			"facebookexternalhit", "twitterbot", "linkedinbot",
			"slurp", "duckduckbot", "ia_archiver",
		},
	}
}

// IsBot checks if the user agent belongs to a known bot.
func (d *BotDetector) IsBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	for _, pattern := range d.patterns {
		if strings.Contains(ua, pattern) {
			return true
		}
	}
	return false
}
