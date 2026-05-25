package seo

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
	"github.com/layer87-labs/webhull/internal/pkg/pages"
)

// Service generates SEO assets: sitemap.xml, robots.txt, meta tags, hreflang.
type Service struct {
	baseURL        string
	siteName       string
	defaultOGImage string
	twitterCard    string
	copyrightStart int
	globalJSONLD   []string
	i18nSvc        *i18n.Service
}

// NewService creates a new SEO service.
func NewService(siteCfg config.SiteIdentity, seoCfg config.SEODefaults, i18nSvc *i18n.Service) *Service {
	twitterCard := seoCfg.DefaultTwitterCard
	if twitterCard == "" {
		twitterCard = "summary_large_image"
	}

	return &Service{
		baseURL:        strings.TrimRight(siteCfg.BaseURL, "/"),
		siteName:       siteCfg.Name,
		defaultOGImage: seoCfg.DefaultOGImage,
		twitterCard:    twitterCard,
		copyrightStart: siteCfg.CopyrightStartYear,
		globalJSONLD:   seoCfg.GlobalJSONLD,
		i18nSvc:        i18nSvc,
	}
}

// BuildMetaTags generates the complete meta tags for a page.
func (s *Service) BuildMetaTags(page *pages.Page) MetaTags {
	canonical := s.baseURL + "/" + page.Slug

	// Gap 1: use seo_title / seo_description with fallback to title / description.
	title := page.SEOTitle
	if title == "" {
		title = page.Title
	}
	desc := page.SEODesc
	if desc == "" {
		desc = page.Description
	}

	ogImage := page.SEO.OGImage
	if ogImage == "" {
		ogImage = s.defaultOGImage
	}
	if ogImage != "" && !strings.HasPrefix(ogImage, "http") {
		ogImage = s.baseURL + ogImage
	}

	// Gap 3: smart og:type default based on template instead of hardcoding "website".
	ogType := page.SEO.OGType
	if ogType == "" {
		switch page.Template {
		case "home", "legal":
			ogType = "website"
		default:
			// contact, default, and all custom templates → article
			ogType = "article"
		}
	}

	// Build hreflang links
	hreflang := make([]HreflangLink, 0, len(page.Alternates)+1)
	for lang, slug := range page.Alternates {
		hreflang = append(hreflang, HreflangLink{
			Lang: lang.String(),
			Href: s.baseURL + "/" + slug,
		})
	}
	// x-default points to the default language version
	if defaultSlug, ok := page.Alternates[s.i18nSvc.Default()]; ok {
		hreflang = append(hreflang, HreflangLink{
			Lang: "x-default",
			Href: s.baseURL + "/" + defaultSlug,
		})
	}

	// Gap 2: merge global JSON-LD with per-page JSON-LD.
	jsonLD := make([]string, 0, len(s.globalJSONLD)+len(page.SEO.JSONLD))
	jsonLD = append(jsonLD, s.globalJSONLD...)
	jsonLD = append(jsonLD, page.SEO.JSONLD...)

	return MetaTags{
		Title:         title,
		Description:   desc,
		Keywords:      page.Keywords,
		CanonicalURL:  canonical,
		OGTitle:       title,
		OGDescription: desc,
		OGImage:       ogImage,
		OGType:        ogType,
		OGUrl:         canonical,
		TwitterCard:   s.twitterCard,
		NoIndex:       page.SEO.NoIndex,
		Hreflang:      hreflang,
		Author:        s.siteName, // Gap 4
		JSONLD:        jsonLD,     // Gap 2
	}
}

// Copyright returns the formatted copyright info.
func (s *Service) Copyright() CopyrightInfo {
	return CopyrightInfo{
		StartYear:   s.copyrightStart,
		CurrentYear: time.Now().Year(),
	}
}

// LanguageSwitchLinks builds the language switcher data for a page.
func (s *Service) LanguageSwitchLinks(page *pages.Page) []LanguageSwitchLink {
	links := make([]LanguageSwitchLink, 0, len(page.Alternates))
	for lang, slug := range page.Alternates {
		links = append(links, LanguageSwitchLink{
			Language: lang,
			URL:      "/" + slug,
			Label:    strings.ToUpper(lang.String()),
			Active:   lang == page.Language,
		})
	}
	return links
}

// ServeSitemap generates and serves sitemap.xml with multilingual support.
func (s *Service) ServeSitemap(allPages []*pages.Page) gin.HandlerFunc {
	return func(c *gin.Context) {
		var sb strings.Builder
		sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">`)

		// Group pages by ID for proper xhtml:link alternates
		pageGroups := make(map[string][]*pages.Page)
		for _, p := range allPages {
			if !p.SEO.NoIndex {
				pageGroups[p.ID] = append(pageGroups[p.ID], p)
			}
		}

		// Gap 5: sort group IDs for deterministic sitemap output.
		groupIDs := make([]string, 0, len(pageGroups))
		for id := range pageGroups {
			groupIDs = append(groupIDs, id)
		}
		sort.Strings(groupIDs)

		for _, id := range groupIDs {
			group := pageGroups[id]
			for _, page := range group {
				sb.WriteString("\n  <url>")
				sb.WriteString(fmt.Sprintf("\n    <loc>%s/%s</loc>", s.baseURL, page.Slug))
				sb.WriteString(fmt.Sprintf("\n    <lastmod>%s</lastmod>", time.Now().Format("2006-01-02")))
				sb.WriteString(fmt.Sprintf("\n    <changefreq>%s</changefreq>", page.SEO.ChangeFreq))
				sb.WriteString(fmt.Sprintf("\n    <priority>%.1f</priority>", page.SEO.Priority))

				// xhtml:link alternates for all language variants
				for _, alt := range group {
					sb.WriteString(fmt.Sprintf("\n    <xhtml:link rel=\"alternate\" hreflang=\"%s\" href=\"%s/%s\" />",
						alt.Language.String(), s.baseURL, alt.Slug))
				}

				sb.WriteString("\n  </url>")
			}
		}

		sb.WriteString("\n</urlset>")

		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusOK, sb.String())
	}
}

// ServeRobotsTxt generates and serves robots.txt.
func (s *Service) ServeRobotsTxt() gin.HandlerFunc {
	return func(c *gin.Context) {
		robots := fmt.Sprintf(`User-agent: *
Allow: /

Sitemap: %s/sitemap.xml

Crawl-delay: 1`, s.baseURL)

		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, robots)
	}
}
