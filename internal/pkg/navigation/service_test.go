package navigation

import (
	"testing"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
)

func testNavConfig() config.NavigationConfig {
	return config.NavigationConfig{
		Header: map[string][]config.NavItemConfig{
			"de": {
				{Slug: "start", Title: "Start", URL: "/start"},
				{
					Slug: "produkte", Title: "Produkte", URL: "/produkte",
					Children: []config.NavItemConfig{
						{Slug: "desk", Title: "Desk", URL: "/desk"},
					},
				},
			},
			"en": {
				{Slug: "home", Title: "Home", URL: "/home"},
				{
					Slug: "products", Title: "Products", URL: "/products",
					Children: []config.NavItemConfig{
						{Slug: "desk", Title: "Desk", URL: "/desk"},
					},
				},
			},
		},
		Footer: map[string][]config.NavItemConfig{
			"de": {
				{Slug: "impressum", Title: "Impressum", URL: "/impressum"},
			},
			"en": {
				{Slug: "imprint", Title: "Imprint", URL: "/imprint"},
			},
		},
	}
}

func TestResolveHeader_ActiveState(t *testing.T) {
	svc := NewService(testNavConfig())
	header := svc.ResolveHeader(i18n.LangDE, "start")

	if !header.Items[0].Active {
		t.Error("home should be active when current slug is 'start'")
	}
	if header.Items[1].Active {
		t.Error("products should not be active")
	}
}

func TestResolveHeader_ParentActiveWhenChildActive(t *testing.T) {
	svc := NewService(testNavConfig())
	header := svc.ResolveHeader(i18n.LangDE, "desk")

	if !header.Items[1].Active {
		t.Error("products parent should be active when child 'desk' is current")
	}
	if len(header.Items[1].Children) == 0 {
		t.Fatal("products should have children")
	}
	if !header.Items[1].Children[0].Active {
		t.Error("desk child should be active")
	}
}

func TestResolveHeader_English(t *testing.T) {
	svc := NewService(testNavConfig())
	header := svc.ResolveHeader(i18n.LangEN, "home")

	if header.Items[0].Title != "Home" {
		t.Errorf("expected title 'Home', got %q", header.Items[0].Title)
	}
	if header.Items[0].URL != "/home" {
		t.Errorf("expected URL '/home', got %q", header.Items[0].URL)
	}
}

func TestResolveFooter(t *testing.T) {
	svc := NewService(testNavConfig())
	footer := svc.ResolveFooter(i18n.LangDE, "start")

	if len(footer.Items) != 1 {
		t.Fatalf("expected 1 footer item, got %d", len(footer.Items))
	}
	if footer.Items[0].Title != "Impressum" {
		t.Errorf("expected 'Impressum', got %q", footer.Items[0].Title)
	}
}

func TestResolveHeader_NoneActive(t *testing.T) {
	svc := NewService(testNavConfig())
	header := svc.ResolveHeader(i18n.LangDE, "unknown-slug")

	for _, item := range header.Items {
		if item.Active {
			t.Errorf("item %q should not be active for unknown slug", item.Title)
		}
	}
}

func TestResolveHeader_UnknownLanguage(t *testing.T) {
	svc := NewService(testNavConfig())
	header := svc.ResolveHeader(i18n.Language("fr"), "start")

	if len(header.Items) != 0 {
		t.Errorf("expected 0 items for unknown language, got %d", len(header.Items))
	}
}

func TestResolveHeader_NestedSlugActivatesParent(t *testing.T) {
	svc := NewService(testNavConfig())
	header := svc.ResolveHeader(i18n.LangDE, "produkte/desk")

	if !header.Items[1].Active {
		t.Error("produkte should be active when current slug is the nested 'produkte/desk'")
	}
	if header.Items[0].Active {
		t.Error("start should not be active")
	}
}

func TestMatchesSlug(t *testing.T) {
	tests := []struct {
		target  string
		current string
		want    bool
	}{
		{"produkte", "produkte", true},
		{"produkte", "produkte/desk", true},
		{"produkte", "produkte/desk/details", true},
		{"produkte/desk", "produkte/desk", true},
		{"produkte", "produkte-alt", false},   // string prefix, not a segment
		{"produkte", "produkte-alt/x", false}, // still not a segment boundary
		{"produkte", "produkt", false},
		{"produkte/desk", "produkte", false}, // child does not activate on parent
		{"", "", true},                       // root nav item on root page
		{"", "start", false},                 // root nav item must not match everything
		{"#", "start", false},                // anchor-only URL never matches
	}

	for _, tt := range tests {
		t.Run(tt.target+"|"+tt.current, func(t *testing.T) {
			if got := matchesSlug(tt.target, tt.current); got != tt.want {
				t.Errorf("matchesSlug(%q, %q) = %v, want %v", tt.target, tt.current, got, tt.want)
			}
		})
	}
}

func TestIsActive_URLFallbackSegmentMatch(t *testing.T) {
	// Items without a slug fall back to the URL path — the same segment
	// rule applies there.
	item := config.NavItemConfig{Title: "Produkte", URL: "/produkte"}

	if !isActive(item, "produkte/desk") {
		t.Error("URL fallback should match a nested slug")
	}
	if isActive(item, "produkte-alt") {
		t.Error("URL fallback must not match a sibling with the same string prefix")
	}
	if isActive(config.NavItemConfig{Title: "Top", URL: "#"}, "produkte") {
		t.Error("anchor-only URL must never be active")
	}
}
