package seo

import (
	"fmt"

	"github.com/layer87-labs/webhull/internal/pkg/i18n"
)

// MetaTags holds all SEO meta information for a single page render.
type MetaTags struct {
	Title         string
	Description   string
	Keywords      string
	CanonicalURL  string
	OGTitle       string
	OGDescription string
	OGImage       string
	OGType        string
	OGUrl         string
	TwitterCard   string
	NoIndex       bool
	Hreflang      []HreflangLink
	// Author is rendered as <meta name="author">. Defaults to the site name.
	Author string
	// JSONLD holds pre-serialised JSON-LD blocks injected as
	// <script type="application/ld+json"> in the page <head>.
	// Contains global blocks (from SEODefaults.GlobalJSONLD) merged with
	// per-page blocks (from PageSEOConfig.JSONLD).
	JSONLD []string
}

// HreflangLink represents a single hreflang alternate link.
type HreflangLink struct {
	Lang string // e.g., "de", "en", "x-default"
	Href string // absolute URL
}

// CopyrightInfo holds dynamic copyright year calculation.
type CopyrightInfo struct {
	StartYear   int
	CurrentYear int
}

// String returns the formatted copyright year string.
func (c CopyrightInfo) String() string {
	if c.StartYear == c.CurrentYear || c.StartYear == 0 {
		return fmt.Sprintf("%d", c.CurrentYear)
	}
	return fmt.Sprintf("%d – %d", c.StartYear, c.CurrentYear)
}

// LanguageSwitchLink holds data for a language switcher entry.
type LanguageSwitchLink struct {
	Language i18n.Language
	URL      string
	Label    string
	Active   bool
}
