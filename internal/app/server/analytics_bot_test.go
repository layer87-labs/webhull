package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/pkg/analytics"
	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/consent"
)

// fakeProvider records every event handed to it so tests can assert on what
// server-side tracking actually dispatched, instead of on a proxy for it.
type fakeProvider struct {
	mu     sync.Mutex
	events []analytics.Event
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) TrackEvent(_ context.Context, event analytics.Event, _, _, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
	return nil
}

func (f *fakeProvider) Close() error { return nil }

func (f *fakeProvider) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

// withFakeAnalytics rewires the server's analytics service onto a fake
// provider, keeping the real BotDetector wired in exactly as production
// does. TrackServerSide is fire-and-forget, so callers must waitForEvents.
func withFakeAnalytics(srv *Server) *fakeProvider {
	fake := &fakeProvider{}
	srv.Analytics = analytics.NewService(zap.NewNop(), srv.Bot.IsBot, fake)
	return fake
}

// waitForEvents polls briefly for TrackServerSide's background goroutine to
// either land the expected count or fail to.
func waitForEvents(t *testing.T, fake *fakeProvider, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fake.count() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := fake.count(); got != want {
		t.Fatalf("events recorded = %d, want %d", got, want)
	}
}

func doWithUA(srv *Server, method, path, userAgent string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	srv.Router().ServeHTTP(rec, req)
	return rec
}

func TestServerSideTracking_SkipsBlackboxProbe(t *testing.T) {
	srv := newTestServer(t, nil)
	fake := withFakeAnalytics(srv)

	rec := doWithUA(srv, http.MethodGet, "/produkte", "Blackbox-Exporter/0.28.0")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: status = %d, want 200", rec.Code)
	}

	time.Sleep(50 * time.Millisecond)
	if got := fake.count(); got != 0 {
		t.Errorf("events recorded for blackbox-exporter probe = %d, want 0", got)
	}
}

func TestServerSideTracking_RealBrowserWithoutConsent(t *testing.T) {
	srv := newTestServer(t, nil)
	fake := withFakeAnalytics(srv)

	rec := doWithUA(srv, http.MethodGet, "/produkte",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: status = %d, want 200", rec.Code)
	}

	waitForEvents(t, fake, 1)
}

func TestServerSideTracking_HeadRequestNotCounted(t *testing.T) {
	srv := newTestServer(t, nil)
	fake := withFakeAnalytics(srv)

	rec := doWithUA(srv, http.MethodHead, "/produkte",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0")
	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD /: status = %d, want 200", rec.Code)
	}

	time.Sleep(50 * time.Millisecond)
	if got := fake.count(); got != 0 {
		t.Errorf("events recorded for HEAD request = %d, want 0", got)
	}
}

func TestServerSideTracking_EmptyUserAgentNotCounted(t *testing.T) {
	srv := newTestServer(t, nil)
	fake := withFakeAnalytics(srv)

	rec := doWithUA(srv, http.MethodGet, "/produkte", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: status = %d, want 200", rec.Code)
	}

	time.Sleep(50 * time.Millisecond)
	if got := fake.count(); got != 0 {
		t.Errorf("events recorded for empty User-Agent = %d, want 0", got)
	}
}

func TestServerSideTracking_ConsentGivenSkipsServerSide(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testSiteConfig(t, nil)
	cfg.Consent = config.ConsentConfig{
		Enabled: true,
		Categories: map[string]config.ConsentCategory{
			"analytics": {Required: false, Default: false},
		},
	}
	srv, err := New(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	fake := withFakeAnalytics(srv)

	state := consent.State{Decided: true, Categories: map[string]bool{"analytics": true}}
	cookieValue, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal consent state: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/produkte", nil)
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0")
	req.AddCookie(&http.Cookie{Name: consent.CookieName, Value: url.QueryEscape(string(cookieValue))})
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: status = %d, want 200", rec.Code)
	}

	time.Sleep(50 * time.Millisecond)
	if got := fake.count(); got != 0 {
		t.Errorf("events recorded with analytics consent given = %d, want 0 (client JS tracks instead)", got)
	}
}
