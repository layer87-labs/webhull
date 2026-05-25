package pages

import "github.com/layer87-labs/webhull/internal/pkg/i18n"

// Page represents a fully resolved page ready for rendering.
type Page struct {
	// ID is the internal identifier (e.g., "contact", "home").
	ID string

	// Template is the template name to render (e.g., "contact", "home").
	Template string

	// Language is the resolved language for this page instance.
	Language i18n.Language

	// Slug is the URL path segment (language-specific).
	Slug string

	// Title is the page title (language-specific).
	Title string

	// SEOTitle overrides Title in <title> and og:title when non-empty (language-specific).
	SEOTitle string

	// Description is the meta description (language-specific).
	Description string

	// SEODesc overrides Description in meta description and og:description when non-empty (language-specific).
	SEODesc string

	// Keywords are the meta keywords (language-specific).
	Keywords string

	// Content holds arbitrary template data (language-specific).
	Content map[string]string

	// Sections holds ordered typed content sections for flexible layout.
	// When non-empty, the template renders these in order instead of using
	// fixed named-key rendering. Populated from typed section markers in the body.
	Sections []Section

	// SEO holds page-level SEO settings.
	SEO PageSEO

	// Alternates maps other languages to their slugs (for hreflang).
	Alternates map[i18n.Language]string
}

// Section represents a single typed content section rendered in declaration order.
// Parsed from <!-- section[type,altbg,id=anchor]: Title HTML --> body markers.
type Section struct {
	// Type controls the CSS layout: "block", "grid", or "services".
	Type string
	// AltBg applies the alt-bg CSS modifier to the outer section element.
	AltBg bool
	// ID is an optional HTML id placed on the outer section element.
	ID string
	// Title is raw HTML rendered in the <h2> section header. Empty means no header.
	Title string
	// Body is raw HTML rendered as the section content.
	Body string
}

// PageSEO holds SEO metadata for a single page.
type PageSEO struct {
	Priority   float64
	ChangeFreq string
	OGImage    string
	OGType     string
	NoIndex    bool
	// JSONLD holds page-specific raw JSON-LD blocks (merged with global blocks at render time).
	JSONLD []string
}

// RenderedPage holds a pre-rendered page's HTML bytes.
type RenderedPage struct {
	Page *Page
	HTML []byte
}
