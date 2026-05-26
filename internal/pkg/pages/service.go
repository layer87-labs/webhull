package pages

import (
	"fmt"

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
