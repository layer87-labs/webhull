package templates

import (
	"testing"

	"github.com/layer87-labs/webhull/internal/pkg/pages"
)

func TestPageData_Content_PluginOverridesPage(t *testing.T) {
	pd := &PageData{
		Page: &pages.Page{
			Content: map[string]string{"fleet": "static-frontmatter-content"},
		},
		PluginContent: map[string]string{"fleet": "plugin-rendered-content"},
	}

	if got := pd.Content("fleet"); got != "plugin-rendered-content" {
		t.Errorf("Content(fleet) = %q, want plugin content to take precedence", got)
	}
	if !pd.HasContent("fleet") {
		t.Error("HasContent(fleet) = false, want true")
	}
}

func TestPageData_Content_FallsBackToPageContent(t *testing.T) {
	pd := &PageData{
		Page: &pages.Page{
			Content: map[string]string{"body": "regular page body"},
		},
	}

	if got := pd.Content("body"); got != "regular page body" {
		t.Errorf("Content(body) = %q, want fallback to Page.Content", got)
	}
	if pd.HasContent("fleet") {
		t.Error("HasContent(fleet) = true, want false — key exists nowhere")
	}
}

func TestPageData_Content_PluginEmptyKeyIsAbsent(t *testing.T) {
	pd := &PageData{
		Page:          &pages.Page{Content: map[string]string{}},
		PluginContent: map[string]string{"fleet": ""},
	}

	if pd.HasContent("fleet") {
		t.Error("HasContent(fleet) = true, want false for an empty plugin fragment (e.g. stale/no data yet)")
	}
	if got := pd.Content("fleet"); got != "" {
		t.Errorf("Content(fleet) = %q, want empty string", got)
	}
}
