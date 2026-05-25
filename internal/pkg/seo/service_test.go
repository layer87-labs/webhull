package seo

import (
	"strings"
	"testing"
	"time"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
	"github.com/layer87-labs/webhull/internal/pkg/pages"
)

func testSEOService() *Service {
	siteCfg := config.SiteIdentity{
		Name:               "Test Site",
		BaseURL:            "https://example.com",
		CopyrightStartYear: 2020,
	}
	seoCfg := config.SEODefaults{
		DefaultOGImage:     "/images/og-default.png",
		DefaultTwitterCard: "summary_large_image",
	}
	i18nSvc := i18n.NewService("de", []string{"de", "en"})
	return NewService(siteCfg, seoCfg, i18nSvc)
}

func testPage() *pages.Page {
	return &pages.Page{
		ID:          "contact",
		Template:    "contact",
		Language:    i18n.LangDE,
		Slug:        "kontakt",
		Title:       "Kontakt",
		Description: "Kontaktieren Sie uns",
		Keywords:    "kontakt, email",
		SEO: pages.PageSEO{
			Priority:   0.8,
			ChangeFreq: "monthly",
		},
		Alternates: map[i18n.Language]string{
			i18n.LangDE: "kontakt",
			i18n.LangEN: "contact",
		},
	}
}

func TestBuildMetaTags_Canonical(t *testing.T) {
	svc := testSEOService()
	meta := svc.BuildMetaTags(testPage())

	expected := "https://example.com/kontakt"
	if meta.CanonicalURL != expected {
		t.Errorf("canonical = %q, want %q", meta.CanonicalURL, expected)
	}
}

func TestBuildMetaTags_OGFields(t *testing.T) {
	svc := testSEOService()
	meta := svc.BuildMetaTags(testPage())

	if meta.OGTitle != "Kontakt" {
		t.Errorf("OGTitle = %q, want 'Kontakt'", meta.OGTitle)
	}
	if meta.OGDescription != "Kontaktieren Sie uns" {
		t.Errorf("OGDescription = %q", meta.OGDescription)
	}
	// contact template should default to og:type=article (Gap 3)
	if meta.OGType != "article" {
		t.Errorf("OGType = %q, want 'article' for contact template", meta.OGType)
	}
	if meta.TwitterCard != "summary_large_image" {
		t.Errorf("TwitterCard = %q", meta.TwitterCard)
	}
}

func TestBuildMetaTags_DefaultOGImage(t *testing.T) {
	svc := testSEOService()
	meta := svc.BuildMetaTags(testPage())

	expected := "https://example.com/images/og-default.png"
	if meta.OGImage != expected {
		t.Errorf("OGImage = %q, want %q", meta.OGImage, expected)
	}
}

func TestBuildMetaTags_CustomOGImage(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.SEO.OGImage = "/images/contact-og.png"
	meta := svc.BuildMetaTags(page)

	expected := "https://example.com/images/contact-og.png"
	if meta.OGImage != expected {
		t.Errorf("OGImage = %q, want %q", meta.OGImage, expected)
	}
}

func TestBuildMetaTags_AbsoluteOGImage(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.SEO.OGImage = "https://cdn.example.com/image.png"
	meta := svc.BuildMetaTags(page)

	if meta.OGImage != "https://cdn.example.com/image.png" {
		t.Errorf("absolute OGImage should not be prefixed: %q", meta.OGImage)
	}
}

func TestBuildMetaTags_Hreflang(t *testing.T) {
	svc := testSEOService()
	meta := svc.BuildMetaTags(testPage())

	// Should have de, en, and x-default
	if len(meta.Hreflang) < 3 {
		t.Fatalf("expected at least 3 hreflang links, got %d", len(meta.Hreflang))
	}

	foundXDefault := false
	for _, link := range meta.Hreflang {
		if link.Lang == "x-default" {
			foundXDefault = true
			if !strings.HasSuffix(link.Href, "/kontakt") {
				t.Errorf("x-default href = %q, should end with /kontakt", link.Href)
			}
		}
	}
	if !foundXDefault {
		t.Error("x-default hreflang link missing")
	}
}

func TestBuildMetaTags_NoIndex(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.SEO.NoIndex = true
	meta := svc.BuildMetaTags(page)
	if !meta.NoIndex {
		t.Error("NoIndex should be true")
	}
}

// --- Gap 1: seo_title / seo_description override ---

func TestBuildMetaTags_SEOTitleOverride(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.SEOTitle = "Layer87 – Kontakt aufnehmen"
	meta := svc.BuildMetaTags(page)

	if meta.Title != "Layer87 – Kontakt aufnehmen" {
		t.Errorf("Title = %q, want SEO override", meta.Title)
	}
	if meta.OGTitle != "Layer87 – Kontakt aufnehmen" {
		t.Errorf("OGTitle = %q, want SEO override", meta.OGTitle)
	}
}

func TestBuildMetaTags_SEOTitleFallback(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	// No SEOTitle set — should fall back to page.Title
	meta := svc.BuildMetaTags(page)

	if meta.Title != "Kontakt" {
		t.Errorf("Title = %q, want fallback to page.Title", meta.Title)
	}
}

func TestBuildMetaTags_SEODescOverride(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.SEODesc = "Nehmen Sie Kontakt mit Layer87 auf – für Softwareentwicklung in Deutschland."
	meta := svc.BuildMetaTags(page)

	if meta.Description != page.SEODesc {
		t.Errorf("Description = %q, want SEO override", meta.Description)
	}
	if meta.OGDescription != page.SEODesc {
		t.Errorf("OGDescription = %q, want SEO override", meta.OGDescription)
	}
}

func TestBuildMetaTags_SEODescFallback(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	meta := svc.BuildMetaTags(page)

	if meta.Description != "Kontaktieren Sie uns" {
		t.Errorf("Description = %q, want fallback to page.Description", meta.Description)
	}
}

// --- Gap 2: JSON-LD ---

func TestBuildMetaTags_GlobalJsonLD(t *testing.T) {
	siteCfg := config.SiteIdentity{Name: "Test Site", BaseURL: "https://example.com"}
	seoCfg := config.SEODefaults{
		GlobalJSONLD: []string{`{"@context":"https://schema.org","@type":"Organization","name":"Test Site"}`},
	}
	svc := NewService(siteCfg, seoCfg, i18n.NewService("de", []string{"de"}))
	meta := svc.BuildMetaTags(testPage())

	if len(meta.JSONLD) != 1 {
		t.Fatalf("JSONLD len = %d, want 1", len(meta.JSONLD))
	}
	if !strings.Contains(meta.JSONLD[0], "Organization") {
		t.Errorf("JSONLD[0] does not contain Organization schema: %q", meta.JSONLD[0])
	}
}

func TestBuildMetaTags_PageJsonLD(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.SEO.JSONLD = []string{`{"@context":"https://schema.org","@type":"ContactPage"}`}
	meta := svc.BuildMetaTags(page)

	if len(meta.JSONLD) != 1 {
		t.Fatalf("JSONLD len = %d, want 1", len(meta.JSONLD))
	}
	if !strings.Contains(meta.JSONLD[0], "ContactPage") {
		t.Errorf("JSONLD[0] does not contain ContactPage: %q", meta.JSONLD[0])
	}
}

func TestBuildMetaTags_MergedJsonLD(t *testing.T) {
	siteCfg := config.SiteIdentity{Name: "Test Site", BaseURL: "https://example.com"}
	seoCfg := config.SEODefaults{
		GlobalJSONLD: []string{`{"@type":"Organization"}`},
	}
	svc := NewService(siteCfg, seoCfg, i18n.NewService("de", []string{"de"}))
	page := testPage()
	page.SEO.JSONLD = []string{`{"@type":"ContactPage"}`}
	meta := svc.BuildMetaTags(page)

	if len(meta.JSONLD) != 2 {
		t.Fatalf("merged JSONLD len = %d, want 2 (global + page)", len(meta.JSONLD))
	}
	// Global block comes first
	if !strings.Contains(meta.JSONLD[0], "Organization") {
		t.Errorf("first block should be global Organization: %q", meta.JSONLD[0])
	}
	if !strings.Contains(meta.JSONLD[1], "ContactPage") {
		t.Errorf("second block should be page ContactPage: %q", meta.JSONLD[1])
	}
}

func TestBuildMetaTags_NoJsonLD(t *testing.T) {
	svc := testSEOService()
	meta := svc.BuildMetaTags(testPage())
	if len(meta.JSONLD) != 0 {
		t.Errorf("expected empty JSONLD, got %d blocks", len(meta.JSONLD))
	}
}

// --- Gap 3: og:type smart defaults ---

func TestBuildMetaTags_OGType_HomeTemplate(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.Template = "home"
	meta := svc.BuildMetaTags(page)
	if meta.OGType != "website" {
		t.Errorf("home template OGType = %q, want 'website'", meta.OGType)
	}
}

func TestBuildMetaTags_OGType_LegalTemplate(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.Template = "legal"
	meta := svc.BuildMetaTags(page)
	if meta.OGType != "website" {
		t.Errorf("legal template OGType = %q, want 'website'", meta.OGType)
	}
}

func TestBuildMetaTags_OGType_DefaultTemplate(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.Template = "default"
	meta := svc.BuildMetaTags(page)
	if meta.OGType != "article" {
		t.Errorf("default template OGType = %q, want 'article'", meta.OGType)
	}
}

func TestBuildMetaTags_OGType_ExplicitOverride(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	page.Template = "home"
	page.SEO.OGType = "product" // explicit override wins
	meta := svc.BuildMetaTags(page)
	if meta.OGType != "product" {
		t.Errorf("explicit OGType = %q, want 'product'", meta.OGType)
	}
}

// --- Gap 4: author meta tag ---

func TestBuildMetaTags_Author(t *testing.T) {
	svc := testSEOService()
	meta := svc.BuildMetaTags(testPage())
	if meta.Author != "Test Site" {
		t.Errorf("Author = %q, want 'Test Site'", meta.Author)
	}
}

// --- Copyright ---

func TestCopyright_SameYear(t *testing.T) {
	info := CopyrightInfo{StartYear: time.Now().Year(), CurrentYear: time.Now().Year()}
	result := info.String()
	if strings.Contains(result, "–") {
		t.Errorf("same year should not contain dash: %q", result)
	}
}

func TestCopyright_Range(t *testing.T) {
	info := CopyrightInfo{StartYear: 2020, CurrentYear: 2025}
	result := info.String()
	if !strings.Contains(result, "2020") || !strings.Contains(result, "2025") {
		t.Errorf("copyright range should contain both years: %q", result)
	}
}

func TestCopyright_ZeroStart(t *testing.T) {
	info := CopyrightInfo{StartYear: 0, CurrentYear: 2025}
	result := info.String()
	if result != "2025" {
		t.Errorf("zero start year should show only current: %q", result)
	}
}

func TestLanguageSwitchLinks(t *testing.T) {
	svc := testSEOService()
	page := testPage()
	links := svc.LanguageSwitchLinks(page)

	if len(links) != 2 {
		t.Fatalf("expected 2 lang switch links, got %d", len(links))
	}

	for _, link := range links {
		if link.Language == i18n.LangDE {
			if !link.Active {
				t.Error("DE should be active for DE page")
			}
			if link.URL != "/kontakt" {
				t.Errorf("DE URL = %q, want '/kontakt'", link.URL)
			}
			if link.Label != "DE" {
				t.Errorf("label = %q, want 'DE'", link.Label)
			}
		}
		if link.Language == i18n.LangEN {
			if link.Active {
				t.Error("EN should not be active for DE page")
			}
			if link.URL != "/contact" {
				t.Errorf("EN URL = %q, want '/contact'", link.URL)
			}
		}
	}
}
