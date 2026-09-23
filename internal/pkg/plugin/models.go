// Package plugin implements webhull's data-source plugin system: declarative
// manifests that fetch data from an external API on a background schedule,
// allowlist the fields that may leave the backend, render them into an HTML
// fragment, and expose the result to page templates via PageData.
//
// A plugin is a directory containing a plugin.yaml manifest and a template
// file referenced by it. Plugins are discovered from a single directory
// (config.SiteConfig.PluginsDir) at startup — no code, no compilation, no
// dynamic linking. Secrets are referenced via ${VAR} and resolved from the
// process environment; literal secrets in the manifest are a hard error.
package plugin

import "time"

// Manifest is the parsed, validated contents of one plugin.yaml.
type Manifest struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name"`

	Source Source `yaml:"source"`
	Select Select `yaml:"select"`
	Render Render `yaml:"render"`
	CSP    CSP    `yaml:"csp"`

	// Enrich optionally fetches one additional resource per item — e.g. a
	// per-vehicle availability calendar — and merges the selected fields
	// into that item before rendering. Still entirely background-refresh
	// driven: N extra requests happen once per refresh cycle, never on the
	// request path.
	Enrich *Enrich `yaml:"enrich,omitempty"`

	// dir is the plugin's directory, used to resolve the render template path.
	// Not part of the YAML schema.
	dir string
}

// Source describes where the plugin's data comes from and how to fetch it.
// v1 supports only HTTP GET against a JSON endpoint; the type is kept as an
// explicit field so future source kinds (e.g. a signed feed, a second HTTP
// method) can be added without breaking existing manifests.
type Source struct {
	// Type selects the fetch strategy. Only "http" is implemented; empty
	// defaults to "http".
	Type string `yaml:"type"`

	URL     string            `yaml:"url"`
	Query   map[string]string `yaml:"query"`
	Headers map[string]string `yaml:"headers"`
	Auth    *Auth             `yaml:"auth,omitempty"`

	// Timeout bounds a single fetch attempt. Default 8s.
	Timeout time.Duration `yaml:"timeout"`

	// RefreshInterval is how often the plugin re-fetches in the background.
	// Requests never trigger a fetch — they only read the last cached
	// result. Default 15m, minimum 30s.
	RefreshInterval time.Duration `yaml:"refreshInterval"`

	// StaleWhileError is how long a previously good result keeps being
	// served after fetches start failing, before the plugin's content is
	// treated as empty. Default 24h.
	StaleWhileError time.Duration `yaml:"staleWhileError"`
}

// Auth describes request authentication. Only HTTP Basic is implemented.
type Auth struct {
	Basic *BasicAuth `yaml:"basic,omitempty"`
}

// BasicAuth holds HTTP Basic credentials. Both values must be written as
// exactly "${VAR}" or "${VAR:default}" in the manifest — a literal is a
// load-time error, same rule as headers. They are expanded after YAML
// parsing, so a password containing quotes, colons or "#" cannot corrupt
// the manifest.
type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Select is the field allowlist applied to every fetched item. Nothing not
// listed here ever reaches a template — this is the plugin system's core
// security boundary between an arbitrary upstream API and the rendered page.
type Select struct {
	// Root is a dot path into the parsed JSON response that resolves to the
	// collection of items to render (e.g. "articles"). Empty means the
	// response body itself is the collection. A collection is either an
	// array, or an object whose values are all objects — the shape of APIs
	// that key records by id ({"17": {...}, "42": {...}}). For the latter,
	// items keep the key order of the response.
	Root string `yaml:"root"`

	// Fields lists dot paths (e.g. "images.outside.medium") extracted from
	// each item. Anything not listed is discarded before rendering.
	Fields []string `yaml:"fields"`
}

// Render describes how allowlisted items become an HTML fragment and where
// that fragment is injected.
type Render struct {
	// Template is the render template's filename, resolved relative to the
	// plugin's own directory (next to plugin.yaml).
	Template string `yaml:"template"`

	Into RenderTarget `yaml:"into"`
}

// RenderTarget names the page and content key the rendered fragment is
// injected into. Page is the page's stable ID (config.PageConfig.ID / the
// "id:" frontmatter field), not the language-specific slug.
type RenderTarget struct {
	Page       string `yaml:"page"`
	ContentKey string `yaml:"contentKey"`
}

// CSP lists the Content-Security-Policy allowances a plugin needs — for
// example the CDN host its images are served from.
type CSP struct {
	ImgSrc []string `yaml:"imgSrc"`
}

// Enrich describes a per-item follow-up fetch: one extra HTTP GET per item
// from the base list, run during the same background refresh cycle. Useful
// for a detail resource the list endpoint doesn't include, e.g. a
// per-vehicle availability calendar.
type Enrich struct {
	Source EnrichSource `yaml:"source"`
	Select EnrichSelect `yaml:"select"`
}

// EnrichSource describes the per-item request. URL and any Query value may
// contain a "{<IDField>}" placeholder, substituted with the item's IDField
// value (URL-query-escaped) before the request is made.
type EnrichSource struct {
	// IDField is the base item field whose value fills the "{...}"
	// placeholder in URL/Query. Defaults to "id". A field holding a list
	// is addressed as "media[].id" (a sub path of each element) or "tags[]"
	// (the elements themselves); its values are joined with commas, for
	// batch endpoints like "?ids=[{media[].id}]".
	IDField string `yaml:"idField"`

	URL     string            `yaml:"url"`
	Query   map[string]string `yaml:"query"`
	Headers map[string]string `yaml:"headers"`
	Auth    *Auth             `yaml:"auth,omitempty"`

	// Timeout bounds a single per-item fetch. Default 8s.
	Timeout time.Duration `yaml:"timeout"`

	// MaxConcurrency bounds how many per-item requests run at once during
	// one refresh cycle. Default 5.
	MaxConcurrency int `yaml:"maxConcurrency"`
}

// EnrichSelect allowlists fields from the per-item response.
//
// Without As, the response is a single JSON object and the selected fields
// are merged into the item. With As, the response holds a collection (at
// Root, same rules as Select.Root); each element is allowlisted and the
// resulting list is stored on the item under the key As.
type EnrichSelect struct {
	Root   string   `yaml:"root"`
	As     string   `yaml:"as"`
	Fields []string `yaml:"fields"`
}
