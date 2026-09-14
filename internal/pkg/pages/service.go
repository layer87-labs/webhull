package pages

import (
	"fmt"
	"strings"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
)

// Service resolves pages from configuration and manages the slug→page lookup.
type Service struct {
	// slugIndex maps "slug" → *Page for O(1) lookup on every request.
	slugIndex map[string]*Page

	// pagesByID maps "pageID:lang" → *Page for cross-referencing.
	pagesByID map[string]*Page

	// startSlugs maps Language → start page slug (for root redirect in multi-page mode).
	startSlugs map[i18n.Language]string

	// rootPages maps Language → root *Page for single-page mode.
	// Populated when the home page declares empty slugs across all languages.
	rootPages map[i18n.Language]*Page

	// allPages holds all resolved pages for iteration (sitemap, pre-rendering).
	allPages []*Page
}

// NewService builds the page index from configuration.
func NewService(pages []config.PageConfig, languages []string) (*Service, error) {
	svc := &Service{
		slugIndex:  make(map[string]*Page),
		pagesByID:  make(map[string]*Page),
		startSlugs: make(map[i18n.Language]string),
		rootPages:  make(map[i18n.Language]*Page),
		allPages:   make([]*Page, 0, len(pages)*len(languages)),
	}

	for _, pageCfg := range pages {
		// Build alternates map for this page
		alternates := make(map[i18n.Language]string)
		for _, lang := range languages {
			if i18nCfg, ok := pageCfg.I18n[lang]; ok {
				alternates[i18n.Language(lang)] = i18nCfg.Slug
			}
		}

		// Create a Page instance per language
		for _, lang := range languages {
			i18nCfg, ok := pageCfg.I18n[lang]
			if !ok {
				return nil, fmt.Errorf("page %q missing i18n for language %q", pageCfg.ID, lang)
			}

			if err := ValidateSlug(i18nCfg.Slug); err != nil {
				return nil, fmt.Errorf("page %q (%s): invalid slug %q: %w", pageCfg.ID, lang, i18nCfg.Slug, err)
			}
			if i18nCfg.Slug == "" && pageCfg.ID != "home" {
				return nil, fmt.Errorf("page %q (%s): empty slug is only allowed for id \"home\" (single-page mode)", pageCfg.ID, lang)
			}

			// The slug index is global — the URL carries no language prefix —
			// so a slug must be unique across ALL languages, not just within one.
			// Root pages ("") are keyed per language and cannot collide here.
			if existing, dup := svc.slugIndex[i18nCfg.Slug]; dup && i18nCfg.Slug != "" {
				return nil, fmt.Errorf("duplicate slug %q: pages %q (%s) and %q (%s) — slugs must be unique across all languages",
					i18nCfg.Slug, existing.ID, existing.Language, pageCfg.ID, lang)
			}

			page := &Page{
				ID:          pageCfg.ID,
				Template:    pageCfg.Template,
				Language:    i18n.Language(lang),
				Slug:        i18nCfg.Slug,
				Title:       i18nCfg.Title,
				SEOTitle:    i18nCfg.SEOTitle,
				Description: i18nCfg.Description,
				SEODesc:     i18nCfg.SEODesc,
				Keywords:    i18nCfg.Keywords,
				Content:     i18nCfg.Content,
				Sections:    convertSections(i18nCfg.Sections),
				SEO: PageSEO{
					Priority:   pageCfg.SEO.Priority,
					ChangeFreq: pageCfg.SEO.ChangeFreq,
					OGImage:    pageCfg.SEO.OGImage,
					OGType:     pageCfg.SEO.OGType,
					NoIndex:    pageCfg.SEO.NoIndex,
					JSONLD:     pageCfg.SEO.JSONLD,
				},
				Alternates: alternates,
			}

			// Default SEO values
			if page.SEO.Priority == 0 {
				page.SEO.Priority = 0.5
			}
			if page.SEO.ChangeFreq == "" {
				page.SEO.ChangeFreq = "monthly"
			}

			svc.slugIndex[i18nCfg.Slug] = page
			svc.pagesByID[fmt.Sprintf("%s:%s", pageCfg.ID, lang)] = page
			svc.allPages = append(svc.allPages, page)

			// Track home page for root routing.
			// Empty slug → single-page mode (render at /); non-empty → multi-page redirect.
			if pageCfg.ID == "home" {
				if i18nCfg.Slug == "" {
					svc.rootPages[i18n.Language(lang)] = page
				} else {
					svc.startSlugs[i18n.Language(lang)] = i18nCfg.Slug
				}
			}
		}
	}

	if len(svc.startSlugs) == 0 && len(svc.rootPages) == 0 {
		return nil, fmt.Errorf(
			"no home page found: declare a page with id \"home\" — " +
				"use non-empty slugs for multi-page mode or empty slugs for single-page mode",
		)
	}

	return svc, nil
}

// Resolve looks up a page by its slug. Returns nil if not found.
func (s *Service) Resolve(slug string) *Page {
	return s.slugIndex[slug]
}

// GetByID looks up a page by its internal ID and language.
func (s *Service) GetByID(pageID string, lang i18n.Language) *Page {
	return s.pagesByID[fmt.Sprintf("%s:%s", pageID, lang)]
}

// StartSlugs returns the start page slugs per language (for root redirect in multi-page mode).
func (s *Service) StartSlugs() map[i18n.Language]string {
	return s.startSlugs
}

// HasRootPages reports whether the site runs in single-page mode.
// Single-page mode is active when the home page has empty slugs across all languages.
func (s *Service) HasRootPages() bool {
	return len(s.rootPages) > 0
}

// RootPage returns the root page for the given language.
// Returns nil if the site is not in single-page mode or the language has no root page.
func (s *Service) RootPage(lang i18n.Language) *Page {
	return s.rootPages[lang]
}

// All returns all resolved pages across all languages.
func (s *Service) All() []*Page {
	return s.allPages
}

// Slugs returns all registered slugs.
func (s *Service) Slugs() []string {
	slugs := make([]string, 0, len(s.slugIndex))
	for slug := range s.slugIndex {
		slugs = append(slugs, slug)
	}
	return slugs
}

// ValidateSlug checks that a slug can be registered as a route.
//
// Slugs may contain "/" to form nested paths ("produkte/desk" → /produkte/desk).
// They must not start or end with a slash, contain whitespace, a query string
// or fragment, or empty / dot segments. The empty slug (root page) is valid
// here; whether a page may use it is decided by the caller.
func ValidateSlug(slug string) error {
	if slug == "" {
		return nil
	}
	if strings.ContainsAny(slug, " \t\r\n") {
		return fmt.Errorf("must not contain whitespace")
	}
	if strings.ContainsAny(slug, "?#") {
		return fmt.Errorf("must not contain \"?\" or \"#\"")
	}
	if strings.HasPrefix(slug, "/") {
		return fmt.Errorf("must not start with \"/\"")
	}
	if strings.HasSuffix(slug, "/") {
		return fmt.Errorf("must not end with \"/\"")
	}
	for _, seg := range strings.Split(slug, "/") {
		switch seg {
		case "":
			return fmt.Errorf("must not contain empty path segments")
		case ".", "..":
			return fmt.Errorf("must not contain %q segments", seg)
		}
	}
	return nil
}

// convertSections converts config.SectionConfig slice to domain Section slice.
func convertSections(cfgSections []config.SectionConfig) []Section {
	if len(cfgSections) == 0 {
		return nil
	}
	sections := make([]Section, len(cfgSections))
	for i, s := range cfgSections {
		sections[i] = Section{
			Type:  s.Type,
			AltBg: s.AltBg,
			ID:    s.ID,
			Title: s.Title,
			Body:  s.Body,
		}
	}
	return sections
}
