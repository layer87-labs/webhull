package pages

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/layer87-labs/webhull/internal/app/templates"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
	"github.com/layer87-labs/webhull/internal/pkg/pages"
)

// render executes a templ component and returns the produced HTML.
func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var sb strings.Builder
	if err := c.Render(context.Background(), &sb); err != nil {
		t.Fatalf("render: %v", err)
	}
	return sb.String()
}

// pageData builds a minimal but complete view model for markup assertions.
func pageData(lang i18n.Language, content map[string]string, sections []pages.Section) *templates.PageData {
	return &templates.PageData{
		Page: &pages.Page{
			ID:       "test",
			Template: "default",
			Language: lang,
			Title:    "Test",
			Content:  content,
			Sections: sections,
		},
	}
}

var typedSections = []pages.Section{
	{Type: "block", ID: "intro", Title: "Intro", Body: "<p>hello</p>"},
	{Type: "grid", AltBg: true, Body: "<p>cards</p>"},
}

func TestTemplates_ExactlyOneMainLandmark(t *testing.T) {
	body := map[string]string{"body": "<p>plain body</p>", "heroLine1": "A", "heroLine2": "B"}

	tests := []struct {
		name string
		html string
	}{
		{"default/typed-sections", render(t, DefaultPage(pageData(i18n.LangDE, body, typedSections)))},
		{"default/plain-body", render(t, DefaultPage(pageData(i18n.LangDE, body, nil)))},
		{"home/typed-sections", render(t, HomePage(pageData(i18n.LangDE, body, typedSections)))},
		{"home/legacy-sections", render(t, HomePage(pageData(i18n.LangDE, map[string]string{
			"heroLine1": "A", "heroLine2": "B", "productsTitle": "P", "productsHTML": "<p>p</p>", "ctaTitle": "C", "ctaText": "go", "ctaLink": "/x",
		}, nil)))},
		{"single/typed-sections", render(t, SinglePage(pageData(i18n.LangDE, body, typedSections)))},
		{"single/raw-body", render(t, SinglePage(pageData(i18n.LangDE, body, nil)))},
		{"contact", render(t, ContactPage(pageData(i18n.LangDE, body, nil)))},
		{"legal", render(t, LegalPage(pageData(i18n.LangDE, body, nil)))},
		{"notfound", render(t, NotFoundPage(pageData(i18n.LangDE, map[string]string{"notFoundTitle": "x"}, nil)))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := strings.Count(tc.html, "<main"); got != 1 {
				t.Errorf("<main> count = %d, want exactly 1", got)
			}
			if got := strings.Count(tc.html, "</main>"); got != 1 {
				t.Errorf("</main> count = %d, want exactly 1", got)
			}
			if !strings.Contains(tc.html, `id="`+templates.MainContentID+`"`) {
				t.Errorf("missing id=%q on the main landmark", templates.MainContentID)
			}
		})
	}
}

func TestDefaultPage_PlainBodyKeepsContentPageClass(t *testing.T) {
	html := render(t, DefaultPage(pageData(i18n.LangDE, map[string]string{"body": "<p>x</p>"}, nil)))
	if !strings.Contains(html, `class="content-page"`) {
		t.Error(`plain body branch lost class="content-page" — site stylesheets depend on it`)
	}
}

func TestDefaultPage_TypedSectionsStillRenderSections(t *testing.T) {
	html := render(t, DefaultPage(pageData(i18n.LangDE, nil, typedSections)))
	if strings.Count(html, "<section") < 2 {
		t.Errorf("typed sections not rendered: %s", html)
	}
	if !strings.Contains(html, `id="intro"`) {
		t.Error("section anchor id lost")
	}
}

func TestSkipLink_IsFirstFocusableElementAndTargetsMain(t *testing.T) {
	html := render(t, DefaultPage(pageData(i18n.LangDE, map[string]string{"body": "<p>x</p>"}, nil)))

	bodyIdx := strings.Index(html, "<body>")
	if bodyIdx < 0 {
		t.Fatal("no <body> in rendered layout")
	}
	rest := html[bodyIdx:]

	attrIdx := strings.Index(rest, `class="skip-link"`)
	if attrIdx < 0 {
		t.Fatal("skip link not rendered")
	}
	// Start of the skip link's own opening tag.
	skipIdx := strings.LastIndex(rest[:attrIdx], "<a ")
	if skipIdx < 0 {
		t.Fatal("skip link is not an anchor element")
	}
	// No other focusable element may precede the skip link.
	for _, tag := range []string{"<a ", "<button", "<input", "<select", "<textarea"} {
		if i := strings.Index(rest, tag); i >= 0 && i < skipIdx {
			t.Errorf("focusable %q at %d precedes the skip link at %d", tag, i, skipIdx)
		}
	}
	if !strings.Contains(rest[skipIdx:attrIdx+200], `href="#`+templates.MainContentID+`"`) {
		t.Errorf("skip link does not target #%s", templates.MainContentID)
	}
	if !strings.Contains(html, "Zum Hauptinhalt springen") {
		t.Error("skip link is missing the German default label")
	}
}

func TestSkipLink_EnglishDefault(t *testing.T) {
	html := render(t, DefaultPage(pageData(i18n.LangEN, map[string]string{"body": "<p>x</p>"}, nil)))
	if !strings.Contains(html, "Skip to main content") {
		t.Error("skip link is missing the English default label")
	}
}

func TestMobileMenuToggle_ARIAAttributes(t *testing.T) {
	html := render(t, DefaultPage(pageData(i18n.LangDE, map[string]string{"body": "<p>x</p>"}, nil)))

	if !strings.Contains(html, `<button type="button" class="mobile-menu-toggle"`) {
		t.Error("mobile menu toggle is not a <button type=\"button\">")
	}
	if !strings.Contains(html, `aria-expanded="false"`) {
		t.Error(`mobile menu toggle is missing aria-expanded="false"`)
	}
	if !strings.Contains(html, `aria-controls="nav-links"`) {
		t.Error(`mobile menu toggle is missing aria-controls="nav-links"`)
	}
	if !strings.Contains(html, `<ul class="nav-links" id="nav-links">`) {
		t.Error("aria-controls target id is missing on the nav list")
	}
}

func TestThemeToggle_LabelIsLocalised(t *testing.T) {
	de := render(t, DefaultPage(pageData(i18n.LangDE, map[string]string{"body": "<p>x</p>"}, nil)))
	if !strings.Contains(de, `aria-label="Design umschalten"`) {
		t.Error("theme toggle keeps an English label on a German page")
	}

	en := render(t, DefaultPage(pageData(i18n.LangEN, map[string]string{"body": "<p>x</p>"}, nil)))
	if !strings.Contains(en, `aria-label="Toggle theme"`) {
		t.Error("theme toggle is missing the English default label")
	}
	if !strings.Contains(en, `<button type="button" class="theme-toggle"`) {
		t.Error("theme toggle is not a <button type=\"button\">")
	}
}
