package templates

import (
	"context"
	"strings"
	"testing"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/consent"
)

func testConsentBanner() *ConsentBannerData {
	return &ConsentBannerData{
		Enabled: true,
		Texts: config.ConsentI18nConfig{
			Title:       "Cookie-Einstellungen",
			Description: "Wir verwenden Cookies.",
			AcceptAll:   "Alle akzeptieren",
			RejectAll:   "Alle ablehnen",
			Customize:   "Anpassen",
			Save:        "Auswahl speichern",
		},
		Categories: []consent.Category{
			{Key: "necessary", Name: "Notwendig", Required: true, Default: true},
			{Key: "analytics", Name: "Analyse", Required: false, Default: false},
		},
	}
}

func TestClientAnalyticsAllowed(t *testing.T) {
	tests := []struct {
		name  string
		state *consent.State
		want  bool
	}{
		{
			name:  "nil state denies",
			state: nil,
			want:  false,
		},
		{
			name:  "undecided denies",
			state: &consent.State{Decided: false, Categories: map[string]bool{"analytics": true}},
			want:  false,
		},
		{
			name:  "decided but rejected denies",
			state: &consent.State{Decided: true, Categories: map[string]bool{"analytics": false}},
			want:  false,
		},
		{
			name:  "category absent denies",
			state: &consent.State{Decided: true, Categories: map[string]bool{"necessary": true}},
			want:  false,
		},
		{
			name:  "decided and accepted allows",
			state: &consent.State{Decided: true, Categories: map[string]bool{"analytics": true}},
			want:  true,
		},
		{
			name:  "bypass accept allows",
			state: &consent.State{Decided: true, Bypassed: true, Categories: map[string]bool{"analytics": true}},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PageData{Consent: tt.state, ConsentConfig: testConsentBanner()}
			if got := pd.ClientAnalyticsAllowed(); got != tt.want {
				t.Errorf("ClientAnalyticsAllowed() = %v, want %v", got, tt.want)
			}
		})
	}

	// Turning consent off is an explicit operator decision — analytics must not
	// silently stop working for those sites.
	t.Run("consent disabled allows", func(t *testing.T) {
		undecided := &consent.State{Decided: false, Categories: map[string]bool{}}

		if pd := (&PageData{Consent: undecided}); !pd.ClientAnalyticsAllowed() {
			t.Error("nil ConsentConfig must allow analytics")
		}

		disabled := testConsentBanner()
		disabled.Enabled = false
		if pd := (&PageData{Consent: undecided, ConsentConfig: disabled}); !pd.ClientAnalyticsAllowed() {
			t.Error("disabled consent must allow analytics")
		}
	})
}

// The Plausible script sends a pageview the moment it loads, so the rendered
// page must not contain it until analytics consent has been accepted.
func TestLayout_PlausibleScriptRequiresConsent(t *testing.T) {
	tests := []struct {
		name      string
		state     *consent.State
		wantInDoc bool
	}{
		{
			name:      "no decision yet",
			state:     &consent.State{Decided: false, Categories: map[string]bool{}},
			wantInDoc: false,
		},
		{
			name:      "analytics rejected",
			state:     &consent.State{Decided: true, Categories: map[string]bool{"necessary": true, "analytics": false}},
			wantInDoc: false,
		},
		{
			name:      "analytics accepted",
			state:     &consent.State{Decided: true, Categories: map[string]bool{"necessary": true, "analytics": true}},
			wantInDoc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PageData{
				Consent:       tt.state,
				ConsentConfig: testConsentBanner(),
				Analytics: AnalyticsData{
					PlausibleEnabled: true,
					PlausibleDomain:  "example.com",
				},
			}

			var sb strings.Builder
			if err := Layout(pd).Render(context.Background(), &sb); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := sb.String()

			if got := strings.Contains(html, `src="/js/script.js"`); got != tt.wantInDoc {
				t.Errorf("plausible script present = %v, want %v", got, tt.wantInDoc)
			}
		})
	}
}

// The collector script is consent-gated on the server too, so it is not even
// fetched before a decision.
func TestLayout_CollectorScriptRequiresConsent(t *testing.T) {
	rejected := &PageData{
		Consent:       &consent.State{Decided: true, Categories: map[string]bool{"analytics": false}},
		ConsentConfig: testConsentBanner(),
		Analytics:     AnalyticsData{CollectorEnabled: true},
	}
	accepted := &PageData{
		Consent:       &consent.State{Decided: true, Categories: map[string]bool{"analytics": true}},
		ConsentConfig: testConsentBanner(),
		Analytics:     AnalyticsData{CollectorEnabled: true},
	}

	var rejectedHTML, acceptedHTML strings.Builder
	if err := Layout(rejected).Render(context.Background(), &rejectedHTML); err != nil {
		t.Fatalf("render rejected: %v", err)
	}
	if err := Layout(accepted).Render(context.Background(), &acceptedHTML); err != nil {
		t.Fatalf("render accepted: %v", err)
	}

	if strings.Contains(rejectedHTML.String(), "/static/js/analytics.js") {
		t.Error("collector script present without analytics consent")
	}
	if !strings.Contains(acceptedHTML.String(), "/static/js/analytics.js") {
		t.Error("collector script missing despite analytics consent")
	}
}

// The banner markup stays in the page after a decision so it can be reopened
// and the decision withdrawn.
func TestLayout_ConsentBannerRemainsReopenable(t *testing.T) {
	tests := []struct {
		name       string
		state      *consent.State
		wantMarkup bool
		wantHidden bool
	}{
		{
			name:       "undecided renders open",
			state:      &consent.State{Decided: false, Categories: map[string]bool{}},
			wantMarkup: true,
			wantHidden: false,
		},
		{
			name:       "decided renders hidden",
			state:      &consent.State{Decided: true, Categories: map[string]bool{"analytics": false}},
			wantMarkup: true,
			wantHidden: true,
		},
		{
			name:       "bypassed renders nothing",
			state:      &consent.State{Decided: true, Bypassed: true, Categories: map[string]bool{"analytics": true}},
			wantMarkup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PageData{Consent: tt.state, ConsentConfig: testConsentBanner()}

			var sb strings.Builder
			if err := Layout(pd).Render(context.Background(), &sb); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := sb.String()

			if got := strings.Contains(html, `id="consent-banner"`); got != tt.wantMarkup {
				t.Fatalf("banner markup present = %v, want %v", got, tt.wantMarkup)
			}
			if !tt.wantMarkup {
				return
			}
			if got := strings.Contains(html, `data-consent-decided="true"`); got != tt.wantHidden {
				t.Errorf("banner hidden = %v, want %v", got, tt.wantHidden)
			}
		})
	}
}

// The dialog needs a programmatic name, a description and a focusable
// container for consent.js to move focus into.
func TestConsentBanner_AccessibleMarkup(t *testing.T) {
	pd := &PageData{
		Consent:       &consent.State{Decided: false, Categories: map[string]bool{}},
		ConsentConfig: testConsentBanner(),
	}

	var sb strings.Builder
	if err := Layout(pd).Render(context.Background(), &sb); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := sb.String()

	for _, want := range []string{
		`aria-labelledby="consent-title"`,
		`aria-describedby="consent-description"`,
		`id="consent-title"`,
		`id="consent-description"`,
		`class="consent-dialog" tabindex="-1"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in rendered dialog", want)
		}
	}
}

// A reopened dialog must show the decision that is actually in effect, not the
// configured defaults.
func TestConsentBannerData_IsChecked(t *testing.T) {
	cfg := testConsentBanner()
	necessary := cfg.Categories[0]
	analytics := cfg.Categories[1]

	t.Run("required is always checked", func(t *testing.T) {
		cfg.State = &consent.State{Decided: true, Categories: map[string]bool{"necessary": false}}
		if !cfg.IsChecked(necessary) {
			t.Error("required category must render checked")
		}
	})

	t.Run("falls back to default before a decision", func(t *testing.T) {
		cfg.State = nil
		if cfg.IsChecked(analytics) {
			t.Error("analytics defaults to off")
		}
		cfg.State = &consent.State{Decided: false, Categories: map[string]bool{"analytics": true}}
		if cfg.IsChecked(analytics) {
			t.Error("undecided state must not be treated as a choice")
		}
	})

	t.Run("stored choice wins over default", func(t *testing.T) {
		cfg.State = &consent.State{Decided: true, Categories: map[string]bool{"analytics": true}}
		if !cfg.IsChecked(analytics) {
			t.Error("accepted analytics must render checked")
		}
		cfg.State = &consent.State{Decided: true, Categories: map[string]bool{"analytics": false}}
		if cfg.IsChecked(analytics) {
			t.Error("rejected analytics must render unchecked")
		}
	})
}

func TestConsentSettingsLabel(t *testing.T) {
	base := func() *PageData {
		return &PageData{
			Consent:       &consent.State{Decided: true, Categories: map[string]bool{}},
			ConsentConfig: testConsentBanner(),
		}
	}

	t.Run("falls back to the banner title", func(t *testing.T) {
		if got := base().ConsentSettingsLabel(); got != "Cookie-Einstellungen" {
			t.Errorf("got %q, want the banner title", got)
		}
	})

	t.Run("explicit label wins", func(t *testing.T) {
		pd := base()
		pd.UI = config.UIStringsConfig{ConsentSettingsLabel: "Cookies verwalten"}
		if got := pd.ConsentSettingsLabel(); got != "Cookies verwalten" {
			t.Errorf("got %q, want the configured label", got)
		}
	})

	t.Run("empty when consent is off", func(t *testing.T) {
		pd := base()
		pd.ConsentConfig = nil
		if got := pd.ConsentSettingsLabel(); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("empty for bypassed audit requests", func(t *testing.T) {
		pd := base()
		pd.Consent.Bypassed = true
		if got := pd.ConsentSettingsLabel(); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}
