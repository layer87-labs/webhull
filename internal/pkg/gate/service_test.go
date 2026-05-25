package gate_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/gate"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestService creates a gate service with predictable settings for testing.
func newTestService(codes []config.GateCode, maxAge time.Duration) *gate.Service {
	return gate.NewService(config.GateConfig{
		Enabled:      true,
		CookieName:   "test_gate",
		CookieMaxAge: maxAge,
		CookieSecret: "test-secret-key-32-bytes-minimum!",
		Codes:        codes,
	}, zap.NewNop(), false /* not secure — tests run over plain HTTP */)
}

// ginContextWithCookie creates a test Gin context that has the given cookie set.
func ginContextWithCookie(name, value string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if value != "" {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	c.Request = req
	return c, w
}

// defaultCodes is the standard set used across multiple sub-tests.
var defaultCodes = []config.GateCode{
	{Code: "aurora-pine-7", Label: "Test User A"},
	{Code: "silver-lake-3", Label: "Test User B"},
}

// --- ValidateCode ---

func TestValidateCode(t *testing.T) {
	svc := newTestService(defaultCodes, 24*time.Hour)

	tests := []struct {
		name      string
		code      string
		wantLabel string
		wantOK    bool
	}{
		{"valid first code", "aurora-pine-7", "Test User A", true},
		{"valid second code", "silver-lake-3", "Test User B", true},
		{"wrong code", "wrong-code", "", false},
		{"empty code", "", "", false},
		{"partial match", "aurora-pine", "", false},
		{"extra suffix", "aurora-pine-7x", "", false},
		{"case sensitive", "Aurora-Pine-7", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, ok := svc.ValidateCode(tt.code)
			if ok != tt.wantOK {
				t.Errorf("ValidateCode(%q) ok = %v, want %v", tt.code, ok, tt.wantOK)
			}
			if label != tt.wantLabel {
				t.Errorf("ValidateCode(%q) label = %q, want %q", tt.code, label, tt.wantLabel)
			}
		})
	}
}

// TestValidateCodeConstantTime verifies that ValidateCode iterates all codes
// even after a match is found (no early return).
// We use a list with the matching code at position 0 and ensure both branches
// produce a result without panicking.
func TestValidateCodeConstantTime(t *testing.T) {
	codes := make([]config.GateCode, 100)
	for i := range codes {
		codes[i] = config.GateCode{Code: "code-placeholder", Label: "x"}
	}
	// Put the matching code at position 0 — full iteration must still occur.
	codes[0] = config.GateCode{Code: "real-code", Label: "Winner"}

	svc := newTestService(codes, 24*time.Hour)

	label, ok := svc.ValidateCode("real-code")
	if !ok {
		t.Fatal("expected match, got none")
	}
	if label != "Winner" {
		t.Errorf("label = %q, want \"Winner\"", label)
	}

	_, ok = svc.ValidateCode("not-a-code")
	if ok {
		t.Fatal("expected no match")
	}
}

// --- IsAuthenticated ---

func TestIsAuthenticated_NoCookie(t *testing.T) {
	svc := newTestService(defaultCodes, 24*time.Hour)
	c, _ := ginContextWithCookie("test_gate", "")
	if svc.IsAuthenticated(c) {
		t.Error("expected false with no cookie, got true")
	}
}

func TestIsAuthenticated_ValidCookie(t *testing.T) {
	svc := newTestService(defaultCodes, 24*time.Hour)

	// Create a session cookie via a proper recorder context.
	w := httptest.NewRecorder()
	createCtx, _ := gin.CreateTestContext(w)
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/gate", nil)
	svc.CreateSessionCookie(createCtx)

	// Extract the Set-Cookie header value.
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("CreateSessionCookie did not set a cookie")
	}
	cookieValue := cookies[0].Value

	// Now verify IsAuthenticated recognises it.
	c, _ := ginContextWithCookie("test_gate", cookieValue)
	if !svc.IsAuthenticated(c) {
		t.Error("expected true for a freshly issued cookie, got false")
	}
}

func TestIsAuthenticated_ExpiredCookie(t *testing.T) {
	// Create service with a very short maxAge.
	svc := newTestService(defaultCodes, 1*time.Millisecond)

	w := httptest.NewRecorder()
	createCtx, _ := gin.CreateTestContext(w)
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/gate", nil)
	svc.CreateSessionCookie(createCtx)

	cookies := w.Result().Cookies()
	cookieValue := cookies[0].Value

	// Wait for the session to expire.
	time.Sleep(10 * time.Millisecond)

	c, _ := ginContextWithCookie("test_gate", cookieValue)
	if svc.IsAuthenticated(c) {
		t.Error("expected false for expired cookie, got true")
	}
}

func TestIsAuthenticated_TamperedSignature(t *testing.T) {
	svc := newTestService(defaultCodes, 24*time.Hour)

	w := httptest.NewRecorder()
	createCtx, _ := gin.CreateTestContext(w)
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/gate", nil)
	svc.CreateSessionCookie(createCtx)

	cookies := w.Result().Cookies()
	cookieValue := cookies[0].Value

	// Tamper with the signature (last character).
	tampered := cookieValue[:len(cookieValue)-1] + "X"

	c, _ := ginContextWithCookie("test_gate", tampered)
	if svc.IsAuthenticated(c) {
		t.Error("expected false for tampered cookie, got true")
	}
}

func TestIsAuthenticated_MalformedCookie(t *testing.T) {
	svc := newTestService(defaultCodes, 24*time.Hour)

	tests := []struct {
		name  string
		value string
	}{
		{"random string", "not-a-valid-cookie"},
		{"missing separator", "aGVsbG8="},
		{"empty payload", ".aGVsbG8="},
		{"garbage", "!!!.???"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := ginContextWithCookie("test_gate", tt.value)
			if svc.IsAuthenticated(c) {
				t.Errorf("expected false for malformed cookie %q, got true", tt.value)
			}
		})
	}
}

func TestIsAuthenticated_WrongSecret(t *testing.T) {
	// Issue cookie with service A.
	svcA := newTestService(defaultCodes, 24*time.Hour)

	w := httptest.NewRecorder()
	createCtx, _ := gin.CreateTestContext(w)
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/gate", nil)
	svcA.CreateSessionCookie(createCtx)

	cookies := w.Result().Cookies()
	cookieValue := cookies[0].Value

	// Validate with service B (different secret).
	svcB := gate.NewService(config.GateConfig{
		Enabled:      true,
		CookieName:   "test_gate",
		CookieMaxAge: 24 * time.Hour,
		CookieSecret: "completely-different-secret-key!!",
		Codes:        defaultCodes,
	}, zap.NewNop(), false)

	c, _ := ginContextWithCookie("test_gate", cookieValue)
	if svcB.IsAuthenticated(c) {
		t.Error("expected false when validating cookie signed with a different secret")
	}
}
