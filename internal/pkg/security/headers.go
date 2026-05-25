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
}

// SecurityHeadersMiddleware sets common security headers including CSP.
func SecurityHeadersMiddleware(cfg SecurityHeadersConfig) gin.HandlerFunc {
	// Pre-build CSPs at init time (static per config).
	csp := buildCSP(cfg)
	arconCSP := buildArconCSP()

	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		if cfg.IsProduction {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Use relaxed CSP for the ARCON experience path (inline scripts + Google Fonts).
		if cfg.ArconPathPrefix != "" && strings.HasPrefix(c.Request.URL.Path, cfg.ArconPathPrefix) {
			c.Header("Content-Security-Policy", arconCSP)
		} else {
			c.Header("Content-Security-Policy", csp)
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

	directives := []string{
		"default-src 'self'",
		"script-src " + scriptSrc,
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"font-src 'self'",
		"connect-src " + connectSrc,
		"frame-src 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}

	return strings.Join(directives, "; ")
}

// buildArconCSP constructs the relaxed CSP used for the ARCON experience pages.
// These pages use inline scripts (canvas animations), inline event handlers,
// and Google Fonts — none of which are compatible with a strict CSP.
func buildArconCSP() string {
	directives := []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline'",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data:",
		"connect-src 'self'",
		"frame-src 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}
	return strings.Join(directives, "; ")
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
