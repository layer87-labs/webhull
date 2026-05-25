package consent

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
)

func testConsentConfig() config.ConsentConfig {
	return config.ConsentConfig{
		Enabled: true,
		Categories: map[string]config.ConsentCategory{
			"necessary": {Required: true, Default: true},
			"analytics": {Required: false, Default: false},
		},
		I18n: map[string]config.ConsentI18nConfig{
			"de": {
				Title:       "Cookies",
				Description: "Wir verwenden Cookies",
				AcceptAll:   "Alle akzeptieren",
				RejectAll:   "Ablehnen",
				Categories:  map[string]string{"necessary": "Notwendig", "analytics": "Analyse"},
			},
			"en": {
				Title:       "Cookies",
				Description: "We use cookies",
				AcceptAll:   "Accept all",
				RejectAll:   "Reject all",
				Categories:  map[string]string{"necessary": "Necessary", "analytics": "Analytics"},
			},
		},
	}
}

func TestService_IsEnabled(t *testing.T) {
	svc := NewService(testConsentConfig())
	if !svc.IsEnabled() {
		t.Error("should be enabled")
	}

	disabled := testConsentConfig()
	disabled.Enabled = false
	svc2 := NewService(disabled)
	if svc2.IsEnabled() {
		t.Error("should be disabled")
	}
}

func TestService_Categories_Localized(t *testing.T) {
	svc := NewService(testConsentConfig())

	de := svc.Categories(i18n.LangDE)
	found := false
	for _, c := range de {
		if c.Key == "necessary" {
			found = true
			if c.Name != "Notwendig" {
				t.Errorf("DE name = %q, want 'Notwendig'", c.Name)
			}
			if !c.Required {
				t.Error("necessary should be required")
			}
		}
	}
	if !found {
		t.Error("necessary category not found")
	}
}

func TestService_Categories_English(t *testing.T) {
	svc := NewService(testConsentConfig())

	en := svc.Categories(i18n.LangEN)
	for _, c := range en {
		if c.Key == "analytics" {
			if c.Name != "Analytics" {
				t.Errorf("EN analytics name = %q, want 'Analytics'", c.Name)
			}
			if c.Required {
				t.Error("analytics should not be required")
			}
		}
	}
}

func TestService_Texts(t *testing.T) {
	svc := NewService(testConsentConfig())
	texts := svc.Texts(i18n.LangEN)
	if texts.Title != "Cookies" {
		t.Errorf("title = %q, want 'Cookies'", texts.Title)
	}
	if texts.AcceptAll != "Accept all" {
		t.Errorf("acceptAll = %q, want 'Accept all'", texts.AcceptAll)
	}
}

func TestService_Texts_German(t *testing.T) {
	svc := NewService(testConsentConfig())
	texts := svc.Texts(i18n.LangDE)
	if texts.AcceptAll != "Alle akzeptieren" {
		t.Errorf("DE acceptAll = %q, want 'Alle akzeptieren'", texts.AcceptAll)
	}
}

func TestService_Texts_UnknownLanguage(t *testing.T) {
	svc := NewService(testConsentConfig())
	texts := svc.Texts(i18n.Language("fr"))
	if texts.Title != "" {
		t.Errorf("unknown lang should return empty texts, got title=%q", texts.Title)
	}
}

func TestState_IsAllowed(t *testing.T) {
	state := &State{
		Decided:    true,
		Categories: map[string]bool{"necessary": true, "analytics": false},
	}
	if !state.IsAllowed("necessary") {
		t.Error("necessary should be allowed")
	}
	if state.IsAllowed("analytics") {
		t.Error("analytics should not be allowed")
	}
	if state.IsAllowed("unknown") {
		t.Error("unknown category should not be allowed")
	}
}

func TestState_IsAllowed_NotDecided(t *testing.T) {
	state := &State{Decided: false, Categories: map[string]bool{"necessary": true}}
	if state.IsAllowed("necessary") {
		t.Error("nothing should be allowed if not decided")
	}
}

func TestState_IsAllowed_AllAccepted(t *testing.T) {
	state := &State{
		Decided:    true,
		Categories: map[string]bool{"necessary": true, "analytics": true, "marketing": true},
	}
	for _, key := range []string{"necessary", "analytics", "marketing"} {
		if !state.IsAllowed(key) {
			t.Errorf("%s should be allowed", key)
		}
	}
}

func TestMiddleware_NoCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewService(testConsentConfig())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler := svc.Middleware()
	handler(c)

	val, exists := c.Get(ContextKey)
	if !exists {
		t.Fatal("consent state should exist in context")
	}
	state := val.(*State)
	if state.Decided {
		t.Error("should not be decided without cookie")
	}
}

func TestMiddleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testConsentConfig()
	cfg.Enabled = false
	svc := NewService(cfg)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler := svc.Middleware()
	handler(c)

	val, _ := c.Get(ContextKey)
	state := val.(*State)
	if !state.Decided {
		t.Error("disabled consent should be decided")
	}
	if !state.Categories["necessary"] {
		t.Error("all categories should be allowed when disabled")
	}
	if !state.Categories["analytics"] {
		t.Error("analytics should be allowed when consent disabled")
	}
}

func TestMiddleware_InvalidCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewService(testConsentConfig())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "not-valid-json"})
	c.Request = req

	handler := svc.Middleware()
	handler(c)

	val, _ := c.Get(ContextKey)
	state := val.(*State)
	if state.Decided {
		t.Error("invalid cookie should result in undecided state")
	}
}

func TestReadCookie_NilContext(t *testing.T) {
	// Verify readCookie handles missing cookies gracefully via middleware
	gin.SetMode(gin.TestMode)
	svc := NewService(testConsentConfig())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	state := svc.readCookie(c)
	if state.Decided {
		t.Error("no cookie should be undecided")
	}
	if state.Categories == nil {
		t.Error("categories map should not be nil")
	}
}

func TestNewService_CategoriesCount(t *testing.T) {
	svc := NewService(testConsentConfig())
	cats := svc.Categories(i18n.LangDE)
	if len(cats) != 2 {
		t.Errorf("expected 2 categories, got %d", len(cats))
	}
}
