package pages

import (
	"testing"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
)

func testPages() []config.PageConfig {
	return []config.PageConfig{
		{
			ID:       "home",
			Template: "home",
			SEO:      config.PageSEOConfig{Priority: 1.0, ChangeFreq: "weekly"},
			I18n: map[string]config.PageI18nConfig{
				"de": {Slug: "start", Title: "Startseite", Description: "DE Home"},
				"en": {Slug: "home", Title: "Home", Description: "EN Home"},
			},
		},
		{
			ID:       "contact",
			Template: "contact",
			SEO:      config.PageSEOConfig{Priority: 0.8, ChangeFreq: "monthly"},
			I18n: map[string]config.PageI18nConfig{
				"de": {Slug: "kontakt", Title: "Kontakt", Description: "DE Kontakt"},
				"en": {Slug: "contact", Title: "Contact", Description: "EN Contact"},
			},
		},
	}
}

func TestNewService_ResolveSlugs(t *testing.T) {
	svc, err := NewService(testPages(), []string{"de", "en"})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	tests := []struct {
		slug     string
		wantID   string
		wantLang i18n.Language
	}{
		{"start", "home", i18n.LangDE},
		{"home", "home", i18n.LangEN},
		{"kontakt", "contact", i18n.LangDE},
		{"contact", "contact", i18n.LangEN},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			page := svc.Resolve(tt.slug)
			if page == nil {
				t.Fatalf("Resolve(%q) returned nil", tt.slug)
			}
			if page.ID != tt.wantID {
				t.Errorf("got ID=%q, want %q", page.ID, tt.wantID)
			}
			if page.Language != tt.wantLang {
				t.Errorf("got Lang=%q, want %q", page.Language, tt.wantLang)
			}
		})
	}
}

func TestNewService_ResolveNotFound(t *testing.T) {
	svc, _ := NewService(testPages(), []string{"de", "en"})
	if page := svc.Resolve("nonexistent"); page != nil {
		t.Errorf("expected nil for unknown slug, got %+v", page)
	}
}

func TestNewService_StartSlugs(t *testing.T) {
	svc, _ := NewService(testPages(), []string{"de", "en"})
	starts := svc.StartSlugs()
	if starts[i18n.LangDE] != "start" {
		t.Errorf("DE start slug = %q, want \"start\"", starts[i18n.LangDE])
	}
	if starts[i18n.LangEN] != "home" {
		t.Errorf("EN start slug = %q, want \"home\"", starts[i18n.LangEN])
	}
}

func TestNewService_Alternates(t *testing.T) {
	svc, _ := NewService(testPages(), []string{"de", "en"})
	page := svc.Resolve("kontakt")
	if page.Alternates[i18n.LangEN] != "contact" {
		t.Errorf("DE kontakt → EN alternate = %q, want \"contact\"", page.Alternates[i18n.LangEN])
	}
}

func TestNewService_GetByID(t *testing.T) {
	svc, _ := NewService(testPages(), []string{"de", "en"})
	page := svc.GetByID("contact", i18n.LangDE)
	if page == nil || page.Slug != "kontakt" {
		t.Errorf("GetByID(contact, de) slug = %v, want \"kontakt\"", page)
	}
}

func TestNewService_AllPages(t *testing.T) {
	svc, _ := NewService(testPages(), []string{"de", "en"})
	all := svc.All()
	if len(all) != 4 {
		t.Errorf("All() returned %d pages, want 4", len(all))
	}
}

func TestNewService_Slugs(t *testing.T) {
	svc, _ := NewService(testPages(), []string{"de", "en"})
	slugs := svc.Slugs()
	if len(slugs) != 4 {
		t.Errorf("Slugs() returned %d slugs, want 4", len(slugs))
	}
}

func TestNewService_NoHomePage(t *testing.T) {
	pages := []config.PageConfig{
		{
			ID:       "about",
			Template: "default",
			I18n: map[string]config.PageI18nConfig{
				"de": {Slug: "ueber-uns", Title: "Über uns"},
			},
		},
	}
	_, err := NewService(pages, []string{"de"})
	if err == nil {
		t.Error("expected error for missing home page, got nil")
	}
}

func TestNewService_MissingI18n(t *testing.T) {
	pages := []config.PageConfig{
		{
			ID:       "home",
			Template: "home",
			I18n: map[string]config.PageI18nConfig{
				"de": {Slug: "start", Title: "Start"},
			},
		},
	}
	_, err := NewService(pages, []string{"de", "en"})
	if err == nil {
		t.Error("expected error for missing EN i18n, got nil")
	}
}

func TestNewService_DefaultSEOValues(t *testing.T) {
	pages := []config.PageConfig{
		{
			ID:       "home",
			Template: "home",
			I18n: map[string]config.PageI18nConfig{
				"de": {Slug: "start", Title: "Start"},
			},
		},
	}
	svc, _ := NewService(pages, []string{"de"})
	page := svc.Resolve("start")
	if page.SEO.Priority != 0.5 {
		t.Errorf("default priority = %f, want 0.5", page.SEO.Priority)
	}
	if page.SEO.ChangeFreq != "monthly" {
		t.Errorf("default changefreq = %q, want \"monthly\"", page.SEO.ChangeFreq)
	}
}
