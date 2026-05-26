package config

import (
	"testing"
	"time"
)

func TestParse_ValidConfig(t *testing.T) {
	yaml := `
site:
  name: "TestSite"
  baseURL: "https://example.com"
i18n:
  defaultLanguage: "de"
  languages: ["de", "en"]
pages:
  - id: "home"
    template: "home"
    i18n:
      de:
        slug: "start"
        title: "Startseite"
        description: "Beschreibung"
      en:
        slug: "home"
        title: "Home"
        description: "Description"
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Site.Name != "TestSite" {
		t.Errorf("expected site name 'TestSite', got %q", cfg.Site.Name)
	}
	if cfg.Site.BaseURL != "https://example.com" {
		t.Errorf("expected baseURL 'https://example.com', got %q", cfg.Site.BaseURL)
	}
	if cfg.I18n.DefaultLanguage != "de" {
		t.Errorf("expected default language 'de', got %q", cfg.I18n.DefaultLanguage)
	}
	if len(cfg.Pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(cfg.Pages))
	}
}

func TestParse_AppliesDefaults(t *testing.T) {
	yaml := `
site:
  name: "TestSite"
  baseURL: "https://example.com"
pages:
  - id: "home"
    template: "home"
    i18n:
      de:
        slug: "start"
        title: "Start"
        description: "Desc"
      en:
        slug: "home"
        title: "Home"
        description: "Desc"
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != "8080" {
		t.Errorf("expected default port '8080', got %q", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Errorf("expected default read timeout 15s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.I18n.DefaultLanguage != "de" {
		t.Errorf("expected default language 'de', got %q", cfg.I18n.DefaultLanguage)
	}
	if cfg.Contact.MaxLinks != 2 {
		t.Errorf("expected default max links 2, got %d", cfg.Contact.MaxLinks)
	}
	if cfg.Mail.Port != 587 {
		t.Errorf("expected default mail port 587, got %d", cfg.Mail.Port)
	}
}

func TestParse_MissingSiteName(t *testing.T) {
	yaml := `
site:
  baseURL: "https://example.com"
pages:
  - id: "home"
    template: "home"
    i18n:
      de:
        slug: "start"
        title: "Start"
        description: "Desc"
      en:
        slug: "home"
        title: "Home"
        description: "Desc"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing site name")
	}
}

func TestParse_MissingBaseURL(t *testing.T) {
	yaml := `
site:
  name: "TestSite"
pages:
  - id: "home"
    template: "home"
    i18n:
      de:
        slug: "start"
        title: "Start"
        description: "Desc"
      en:
        slug: "home"
        title: "Home"
        description: "Desc"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing baseURL")
	}
}

func TestParse_NoPages(t *testing.T) {
	yaml := `
site:
  name: "TestSite"
  baseURL: "https://example.com"
pages: []
`
	// Pages can now be empty — they are loaded separately via content.Load()
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Pages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(cfg.Pages))
	}
}

func TestParse_MissingI18nForLanguage(t *testing.T) {
	yaml := `
site:
  name: "TestSite"
  baseURL: "https://example.com"
i18n:
  defaultLanguage: "de"
  languages: ["de", "en"]
pages:
  - id: "home"
    template: "home"
    i18n:
      de:
        slug: "start"
        title: "Start"
        description: "Desc"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing i18n config")
	}
}

func TestParse_DuplicateSlugs(t *testing.T) {
	yaml := `
site:
  name: "TestSite"
  baseURL: "https://example.com"
i18n:
  defaultLanguage: "de"
  languages: ["de", "en"]
pages:
  - id: "home"
    template: "home"
    i18n:
      de:
        slug: "start"
        title: "Start"
        description: "Desc"
      en:
        slug: "home"
        title: "Home"
        description: "Desc"
  - id: "other"
    template: "default"
    i18n:
      de:
        slug: "start"
        title: "Other"
        description: "Desc"
      en:
        slug: "other"
        title: "Other"
        description: "Desc"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for duplicate slugs")
	}
}

func TestParse_EmptySlug(t *testing.T) {
	// Empty slug is allowed for id: "home" — activates single-page mode.
	yaml := `
site:
  name: "TestSite"
  baseURL: "https://example.com"
i18n:
  defaultLanguage: "de"
  languages: ["de", "en"]
pages:
  - id: "home"
    template: "single"
    i18n:
      de:
        slug: ""
        title: "Start"
        description: "Desc"
      en:
        slug: ""
        title: "Home"
        description: "Desc"
`
	_, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("expected no error for home page with empty slug (single-page mode), got: %v", err)
	}
}

func TestParse_EmptySlugNonHome(t *testing.T) {
	// Empty slug for a non-home page is still an error.
	yaml := `
site:
  name: "TestSite"
  baseURL: "https://example.com"
i18n:
  defaultLanguage: "de"
  languages: ["de"]
pages:
  - id: "home"
    template: "single"
    i18n:
      de:
        slug: ""
        title: "Start"
        description: "Desc"
  - id: "about"
    i18n:
      de:
        slug: ""
        title: "About"
        description: "Desc"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for non-home page with empty slug")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	yaml := `{{{invalid yaml`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/file.yaml", "")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
