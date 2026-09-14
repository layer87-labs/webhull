package content

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestParseFile_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "desk.html")
	os.WriteFile(file, []byte("---\nid: desk\ntemplate: default\ntitle: \"Layer87 Desk\"\nheroTitle: \"Layer87 Desk\"\nheroSubtitle: \"Groupware\"\n---\n\n<div class=\"content-section\">\n  <h2>Hello World</h2>\n</div>\n"), 0o644)

	cf, err := parseFile(file, "de")
	if err != nil {
		t.Fatal(err)
	}
	if cf.Slug != "desk" {
		t.Errorf("slug = %q, want desk", cf.Slug)
	}
	if cf.Lang != "de" {
		t.Errorf("lang = %q, want de", cf.Lang)
	}
	if stringMeta(cf.Meta, "id", "") != "desk" {
		t.Errorf("id = %q, want desk", stringMeta(cf.Meta, "id", ""))
	}
	if stringMeta(cf.Meta, "heroTitle", "") != "Layer87 Desk" {
		t.Errorf("heroTitle = %q", stringMeta(cf.Meta, "heroTitle", ""))
	}
	if cf.Body == "" || cf.Body[:4] != "<div" {
		t.Errorf("body should start with <div, got %q", cf.Body[:20])
	}
}

func TestParseFile_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "simple.html")
	os.WriteFile(file, []byte("<h1>Hello</h1>\n<p>No frontmatter</p>\n"), 0o644)

	cf, err := parseFile(file, "en")
	if err != nil {
		t.Fatal(err)
	}
	if cf.Slug != "simple" {
		t.Errorf("slug = %q, want simple", cf.Slug)
	}
	if len(cf.Meta) != 0 {
		t.Errorf("meta should be empty, got %v", cf.Meta)
	}
	if cf.Body == "" {
		t.Error("body should not be empty")
	}
}

func TestLoad_FullStructure(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "de"), 0o755)
	os.MkdirAll(filepath.Join(dir, "en"), 0o755)

	os.WriteFile(filepath.Join(dir, "de", "start.html"), []byte("---\nid: home\ntemplate: home\ntitle: \"Startseite\"\ndescription: \"Willkommen\"\nheroLine1: \"Digitale Infrastruktur\"\n---\n\n<p>Deutsche Startseite</p>\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "en", "home.html"), []byte("---\nid: home\ntemplate: home\ntitle: \"Home\"\ndescription: \"Welcome\"\nheroLine1: \"Digital Infrastructure\"\n---\n\n<p>English home page</p>\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "de", "kontakt.html"), []byte("---\nid: contact\ntemplate: contact\ntitle: \"Kontakt\"\n---\n\n<p>Kontaktformular</p>\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "en", "contact.html"), []byte("---\nid: contact\ntemplate: contact\ntitle: \"Contact\"\n---\n\n<p>Contact form</p>\n"), 0o644)

	logger := zap.NewNop()
	pages, err := Load(dir, []string{"de", "en"}, logger)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}

	// Find home page
	var found bool
	for _, p := range pages {
		if p.ID == "home" {
			found = true
			if p.Template != "home" {
				t.Errorf("home template = %q, want home", p.Template)
			}
			if _, ok := p.I18n["de"]; !ok {
				t.Error("home missing DE")
			}
			if _, ok := p.I18n["en"]; !ok {
				t.Error("home missing EN")
			}
			if p.I18n["de"].Content["heroLine1"] != "Digitale Infrastruktur" {
				t.Errorf("de heroLine1 = %q", p.I18n["de"].Content["heroLine1"])
			}
			if p.I18n["en"].Content["heroLine1"] != "Digital Infrastructure" {
				t.Errorf("en heroLine1 = %q", p.I18n["en"].Content["heroLine1"])
			}
			if p.I18n["de"].Content["body"] == "" {
				t.Error("de body should not be empty")
			}
		}
	}
	if !found {
		t.Error("home page not found")
	}
}

func TestLoad_CustomKeysInContent(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "de"), 0o755)

	os.WriteFile(filepath.Join(dir, "de", "test.html"), []byte("---\nid: test\ntemplate: default\ntitle: \"Test\"\ncustomField: \"custom value\"\nanotherField: \"another\"\n---\n\n<p>body</p>\n"), 0o644)

	logger := zap.NewNop()
	pages, err := Load(dir, []string{"de"}, logger)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}

	content := pages[0].I18n["de"].Content
	if content["customField"] != "custom value" {
		t.Errorf("customField = %q", content["customField"])
	}
	if content["anotherField"] != "another" {
		t.Errorf("anotherField = %q", content["anotherField"])
	}
	if _, ok := content["id"]; ok {
		t.Error("reserved key 'id' should not be in content")
	}
	if _, ok := content["template"]; ok {
		t.Error("reserved key 'template' should not be in content")
	}
	if _, ok := content["title"]; ok {
		t.Error("reserved key 'title' should not be in content")
	}
}

func TestLoad_EmptyDir(t *testing.T) {
	logger := zap.NewNop()
	pages, err := Load("", []string{"de"}, logger)
	if err != nil {
		t.Fatal(err)
	}
	if pages != nil {
		t.Error("expected nil for empty contentDir")
	}
}

func TestLoad_SlugFrontmatterWithSlash(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "de"), 0o755)

	os.WriteFile(filepath.Join(dir, "de", "start.html"), []byte("---\nid: home\ntemplate: home\ntitle: \"Start\"\n---\n<p>start</p>\n"), 0o644)
	// Flat file, nested URL: the slug key overrides the filename-derived slug.
	os.WriteFile(filepath.Join(dir, "de", "produkte-desk.html"), []byte("---\nid: product-desk\ntemplate: default\ntitle: \"Desk\"\nslug: produkte/desk\n---\n<p>desk</p>\n"), 0o644)

	pages, err := Load(dir, []string{"de"}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}

	var got string
	for _, p := range pages {
		if p.ID == "product-desk" {
			got = p.I18n["de"].Slug
			if _, leaked := p.I18n["de"].Content["slug"]; leaked {
				t.Error("slug is a reserved key and must not leak into the content map")
			}
		}
	}
	if got != "produkte/desk" {
		t.Errorf("slug = %q, want \"produkte/desk\"", got)
	}
}

func TestLoad_SlugFrontmatterNullKeepsFilename(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "de"), 0o755)
	os.WriteFile(filepath.Join(dir, "de", "start.html"), []byte("---\nid: home\ntemplate: home\ntitle: \"Start\"\nslug:\n---\n<p>start</p>\n"), 0o644)

	pages, err := Load(dir, []string{"de"}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	if pages[0].I18n["de"].Slug != "start" {
		t.Errorf("slug = %q, want filename-derived \"start\" for a null slug key", pages[0].I18n["de"].Slug)
	}
}

func TestLoad_SubdirectoryIgnored(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "de", "produkte"), 0o755)

	os.WriteFile(filepath.Join(dir, "de", "start.html"), []byte("---\nid: home\ntemplate: home\ntitle: \"Start\"\n---\n<p>start</p>\n"), 0o644)
	// A file inside a subdirectory of a language directory is never loaded —
	// nested URLs come from the slug key, not from the directory tree.
	os.WriteFile(filepath.Join(dir, "de", "produkte", "desk.html"), []byte("---\nid: desk\ntemplate: default\ntitle: \"Desk\"\n---\n<p>desk</p>\n"), 0o644)

	pages, err := Load(dir, []string{"de"}, zap.NewNop())
	if err != nil {
		t.Fatalf("a subdirectory must not fail loading: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page (subdirectory ignored), got %d", len(pages))
	}
	if pages[0].ID != "home" {
		t.Errorf("loaded page = %q, want home", pages[0].ID)
	}
}
