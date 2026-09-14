package pages

import (
	"strings"
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

func TestNewService_NestedSlug(t *testing.T) {
	pages := append(testPages(), config.PageConfig{
		ID:       "product-desk",
		Template: "default",
		I18n: map[string]config.PageI18nConfig{
			"de": {Slug: "produkte/desk", Title: "Desk"},
			"en": {Slug: "products/desk", Title: "Desk"},
		},
	})
	svc, err := NewService(pages, []string{"de", "en"})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	page := svc.Resolve("produkte/desk")
	if page == nil {
		t.Fatal("Resolve(\"produkte/desk\") returned nil")
	}
	if page.ID != "product-desk" || page.Language != i18n.LangDE {
		t.Errorf("got ID=%q lang=%q, want product-desk/de", page.ID, page.Language)
	}
	if page.Alternates[i18n.LangEN] != "products/desk" {
		t.Errorf("EN alternate = %q, want \"products/desk\"", page.Alternates[i18n.LangEN])
	}
	if svc.GetByID("product-desk", i18n.LangEN).Slug != "products/desk" {
		t.Error("GetByID(product-desk, en) should resolve the nested EN slug")
	}

	found := false
	for _, slug := range svc.Slugs() {
		if slug == "produkte/desk" {
			found = true
		}
	}
	if !found {
		t.Error("Slugs() should contain the nested slug for route registration")
	}
}

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		slug    string
		wantErr string // empty = valid
	}{
		{"", ""},
		{"start", ""},
		{"ueber-uns", ""},
		{"produkte/desk", ""},
		{"a/b/c", ""},
		{"/produkte/desk", `must not start with "/"`},
		{"produkte/desk/", `must not end with "/"`},
		{"produkte//desk", "empty path segments"},
		{"produkte/../desk", `".."`},
		{"produkte/./desk", `"."`},
		{"..", `".."`},
		{"produkte desk", "whitespace"},
		{"produkte\tdesk", "whitespace"},
		{"produkte?x=1", `"?"`},
		{"produkte#top", `"#"`},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			err := ValidateSlug(tt.slug)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateSlug(%q) = %v, want nil", tt.slug, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateSlug(%q) = nil, want error containing %q", tt.slug, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ValidateSlug(%q) = %q, want it to contain %q", tt.slug, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewService_InvalidSlugRejected(t *testing.T) {
	pages := append(testPages(), config.PageConfig{
		ID:       "broken",
		Template: "default",
		I18n: map[string]config.PageI18nConfig{
			"de": {Slug: "/produkte/desk", Title: "Desk"},
			"en": {Slug: "products/desk", Title: "Desk"},
		},
	})
	_, err := NewService(pages, []string{"de", "en"})
	if err == nil {
		t.Fatal("expected error for slug with leading slash, got nil")
	}
	for _, want := range []string{`page "broken"`, "(de)", `"/produkte/desk"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

func TestNewService_DuplicateSlugAcrossLanguages(t *testing.T) {
	// The URL carries no language prefix, so "desk" in DE and "desk" in EN
	// would fight over the same route. That must fail at startup.
	pages := append(testPages(), config.PageConfig{
		ID:       "desk",
		Template: "default",
		I18n: map[string]config.PageI18nConfig{
			"de": {Slug: "desk", Title: "Desk"},
			"en": {Slug: "desk", Title: "Desk"},
		},
	})
	_, err := NewService(pages, []string{"de", "en"})
	if err == nil {
		t.Fatal("expected error for duplicate slug across languages, got nil")
	}
	if !strings.Contains(err.Error(), `duplicate slug "desk"`) {
		t.Errorf("error = %q, want it to mention the duplicate slug", err.Error())
	}
}

func TestNewService_DuplicateSlugWithinLanguage(t *testing.T) {
	pages := append(testPages(), config.PageConfig{
		ID:       "other",
		Template: "default",
		I18n: map[string]config.PageI18nConfig{
			"de": {Slug: "kontakt", Title: "Noch ein Kontakt"},
			"en": {Slug: "other", Title: "Other"},
		},
	})
	_, err := NewService(pages, []string{"de", "en"})
	if err == nil {
		t.Fatal("expected error for duplicate slug within a language, got nil")
	}
	for _, want := range []string{`duplicate slug "kontakt"`, `"contact" (de)`, `"other" (de)`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

func TestNewService_EmptySlugOnlyForHome(t *testing.T) {
	pages := []config.PageConfig{
		{
			ID:       "home",
			Template: "single",
			I18n: map[string]config.PageI18nConfig{
				"de": {Slug: "", Title: "Start"},
				"en": {Slug: "", Title: "Home"},
			},
		},
		{
			ID:       "about",
			Template: "default",
			I18n: map[string]config.PageI18nConfig{
				"de": {Slug: "", Title: "Über uns"},
				"en": {Slug: "about", Title: "About"},
			},
		},
	}
	_, err := NewService(pages, []string{"de", "en"})
	if err == nil {
		t.Fatal("expected error for empty slug on non-home page, got nil")
	}
	if !strings.Contains(err.Error(), `page "about" (de)`) {
		t.Errorf("error = %q, want it to name the page", err.Error())
	}

	// Two root pages (one per language) are fine — that is single-page mode.
	svc, err := NewService(pages[:1], []string{"de", "en"})
	if err != nil {
		t.Fatalf("single-page mode should be valid, got %v", err)
	}
	if !svc.HasRootPages() {
		t.Error("expected single-page mode")
	}
}
