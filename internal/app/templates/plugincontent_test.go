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

func TestPageData_Content_InlineMarkerSubstitution(t *testing.T) {
	pd := &PageData{
		Page: &pages.Page{
			Content: map[string]string{
				"body": "<main><h1>Vermietung</h1><!-- plugin: fleet --></main>",
			},
		},
		PluginContent: map[string]string{"fleet": "<div class=\"fleet-grid\">5 cars</div>"},
	}

	got := pd.Content("body")
	want := "<main><h1>Vermietung</h1><div class=\"fleet-grid\">5 cars</div></main>"
	if got != want {
		t.Errorf("Content(body) = %q, want %q", got, want)
	}
}

func TestPageData_Content_InlineMarkerNoPluginDataYet(t *testing.T) {
	pd := &PageData{
		Page: &pages.Page{
			Content: map[string]string{
				"body": "<main><!-- plugin: fleet --></main>",
			},
		},
		// No PluginContent set — e.g. the plugin's first fetch hasn't
		// completed yet. The marker must not leak into the rendered page.
	}

	got := pd.Content("body")
	if got != "<main></main>" {
		t.Errorf("Content(body) = %q, want marker stripped to empty string", got)
	}
}

func TestPageData_Content_NoMarkerFastPathUnaffected(t *testing.T) {
	pd := &PageData{
		Page: &pages.Page{
			Content: map[string]string{"body": "<p>plain content, no markers</p>"},
		},
	}

	if got := pd.Content("body"); got != "<p>plain content, no markers</p>" {
		t.Errorf("Content(body) = %q, want unchanged", got)
	}
}
