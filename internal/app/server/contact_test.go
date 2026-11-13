package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/forms"
	"github.com/layer87-labs/webhull/internal/pkg/security"
)

// sentMail is one message handed to the fake mailer.
type sentMail struct {
	to, subject, body string
}

// fakeMailer records mails instead of talking SMTP.
type fakeMailer struct {
	mu   sync.Mutex
	sent []sentMail
}

func (m *fakeMailer) record(to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, sentMail{to: to, subject: subject, body: body})
	return nil
}

func (m *fakeMailer) Send(to, subject, body string) error     { return m.record(to, subject, body) }
func (m *fakeMailer) SendHTML(to, subject, body string) error { return m.record(to, subject, body) }
func (m *fakeMailer) SendMultipart(to, subject, plain, _ string) error {
	return m.record(to, subject, plain)
}

func (m *fakeMailer) mails() []sentMail {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]sentMail(nil), m.sent...)
}

// waitForMails waits until at least want mails were recorded.
func (m *fakeMailer) waitForMails(t *testing.T, want int) []sentMail {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := m.mails(); len(got) >= want {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected %d mails, got %d", want, len(m.mails()))
	return nil
}

const ownerAddress = "owner@example.test"

// contactConfig enables the contact form with an auto-reply template that
// asks for every field placeholder, the way older templates did.
func contactConfig(t *testing.T) *config.SiteConfig {
	t.Helper()
	cfg := testSiteConfig(t, nil)
	cfg.Contact = config.ContactConfig{
		Enabled:    true,
		Recipients: []string{ownerAddress},
		MaxLinks:   2,
		RateLimit:  config.RateLimitConfig{Requests: 3, Window: 15 * time.Minute},
	}
	cfg.Mail = config.MailConfig{
		From: "web@example.test",
		Templates: map[string]config.MailTemplateConfig{
			"de": {Subject: "Danke", Body: "Hallo {{NAME}}: {{SUBJECT}} / {{MESSAGE}} Ref {{CONTACT_ID}}"},
		},
	}
	return cfg
}

// newContactServer builds a server for cfg whose form service sends through
// a fake mailer.
func newContactServer(t *testing.T, cfg *config.SiteConfig) (*Server, *fakeMailer) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	srv, err := New(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	mailer := &fakeMailer{}
	limiter := security.NewRateLimiter(security.RateLimitConfig{
		Limit:  cfg.Contact.RateLimit.Requests,
		Window: cfg.Contact.RateLimit.Window,
	}, zap.NewNop())
	srv.Forms = forms.NewService(cfg.Contact, cfg.Mail, mailer, limiter, zap.NewNop())
	return srv, mailer
}

func contactBody(fields map[string]string) string {
	b, _ := json.Marshal(map[string]any{"fields": fields})
	return string(b)
}

func validFields() map[string]string {
	return map[string]string{
		"name":    "Erika Beispiel",
		"email":   "erika@example.test",
		"subject": "Anfrage",
		"message": "Bitte um Rückruf.",
	}
}

// postContact sends a contact submission from peer 192.0.2.1 (the httptest
// default) with an optional X-Forwarded-For header.
func postContact(srv *Server, body, forwardedFor string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	srv.Router().ServeHTTP(rec, req)
	return rec
}

func TestContact_ForwardedForIgnoredWithoutTrustedProxies(t *testing.T) {
	srv, _ := newContactServer(t, contactConfig(t))
	body := contactBody(validFields())

	forwarded := []string{"198.51.100.1", "198.51.100.2", "198.51.100.3", "198.51.100.4"}
	codes := make([]int, 0, len(forwarded))
	for _, xff := range forwarded {
		codes = append(codes, postContact(srv, body, xff).Code)
	}

	want := []int{200, 200, 200, 429}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("status codes = %v, want %v", codes, want)
		}
	}
}

func TestContact_AutoReplyCarriesNoSubmittedText(t *testing.T) {
	srv, mailer := newContactServer(t, contactConfig(t))
	fields := validFields()
	fields["name"] = "Visit shop dot example dot test"
	fields["message"] = "Cheap offers today only"

	rec := postContact(srv, contactBody(fields), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var resp forms.ContactResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	var reply *sentMail
	for _, m := range mailer.waitForMails(t, 2) {
		if m.to == "erika@example.test" {
			reply = &m
		}
	}
	if reply == nil {
		t.Fatal("no auto-reply sent to the visitor")
	}
	want := "Hallo :  /  Ref " + resp.ContactID
	if reply.body != want {
		t.Fatalf("auto-reply body = %q, want %q", reply.body, want)
	}
}

func TestContact_OversizedBodyRejected(t *testing.T) {
	srv, mailer := newContactServer(t, contactConfig(t))
	fields := validFields()
	fields["message"] = strings.Repeat("Text ", 8<<10)

	rec := postContact(srv, contactBody(fields), "")
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
	time.Sleep(50 * time.Millisecond)
	if n := len(mailer.mails()); n != 0 {
		t.Fatalf("%d mails sent for a rejected submission", n)
	}
}

func TestContact_OverlongFieldRejected(t *testing.T) {
	srv, mailer := newContactServer(t, contactConfig(t))
	fields := validFields()
	fields["subject"] = strings.Repeat("Anfrage ", 40)

	rec := postContact(srv, contactBody(fields), "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var resp forms.ContactResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Message != "field too long: subject" {
		t.Fatalf("message = %q", resp.Message)
	}
	time.Sleep(50 * time.Millisecond)
	if n := len(mailer.mails()); n != 0 {
		t.Fatalf("%d mails sent for a rejected submission", n)
	}
}

func TestContact_EmailWithLineBreakRejected(t *testing.T) {
	for _, email := range []string{
		"erika@example.test\r\n",
		"erika@example.test\nBcc: other@example.test",
		"\nerika@example.test",
	} {
		srv, mailer := newContactServer(t, contactConfig(t))
		fields := validFields()
		fields["email"] = email

		rec := postContact(srv, contactBody(fields), "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("email %q: status = %d, want 400", email, rec.Code)
		}
		time.Sleep(50 * time.Millisecond)
		if n := len(mailer.mails()); n != 0 {
			t.Fatalf("email %q: %d mails sent", email, n)
		}
	}
}

func TestContact_ForwardedForReadFromTrustedProxy(t *testing.T) {
	cfg := contactConfig(t)
	cfg.Server.TrustedProxies = []string{"192.0.2.0/24"}
	srv, _ := newContactServer(t, cfg)
	body := contactBody(validFields())

	// Three submissions use up the limit of 198.51.100.1 only.
	for i := 0; i < 3; i++ {
		if code := postContact(srv, body, "198.51.100.1").Code; code != http.StatusOK {
			t.Fatalf("submission %d: status = %d", i+1, code)
		}
	}
	if code := postContact(srv, body, "198.51.100.1").Code; code != http.StatusTooManyRequests {
		t.Fatalf("fourth submission from the same client: status = %d, want 429", code)
	}
	if code := postContact(srv, body, "198.51.100.2").Code; code != http.StatusOK {
		t.Fatalf("other client behind the proxy: status = %d, want 200", code)
	}
}

func TestServer_InvalidTrustedProxyFailsStartup(t *testing.T) {
	cfg := testSiteConfig(t, nil)
	cfg.Server.TrustedProxies = []string{"not-a-cidr"}
	if _, err := New(cfg, zap.NewNop()); err == nil {
		t.Fatal("server.New accepted an invalid trusted proxy")
	}
}

func TestContact_AutoReplyGlobalCap(t *testing.T) {
	cfg := contactConfig(t)
	cfg.Contact.AutoReplyRateLimit = config.RateLimitConfig{Requests: 2, Window: time.Hour}
	srv, mailer := newContactServer(t, cfg)

	for i, addr := range []string{"a@example.test", "b@example.test", "c@example.test"} {
		fields := validFields()
		fields["email"] = addr
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(contactBody(fields)))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113." + string(rune('1'+i)) + ":4000"
		srv.Router().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("submission %s: status = %d", addr, rec.Code)
		}
	}

	// Three owner notifications plus two auto-replies.
	mailer.waitForMails(t, 5)
	time.Sleep(50 * time.Millisecond)
	mails := mailer.mails()
	replies, notifications := 0, 0
	for _, m := range mails {
		if m.to == ownerAddress {
			notifications++
		} else {
			replies++
		}
	}
	if notifications != 3 || replies != 2 {
		t.Fatalf("notifications = %d, auto-replies = %d, want 3 and 2", notifications, replies)
	}
}

// postContactFrom sends a contact submission from peer with the given
// forwarding headers (empty values are omitted).
func postContactFrom(srv *Server, peer, forwardedFor, realIP string) int {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(contactBody(validFields())))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = peer
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
	}
	srv.Router().ServeHTTP(rec, req)
	return rec.Code
}

// assertFourthLimited sends four submissions whose forwarding headers differ
// and expects the fourth to be rate limited: the headers must not have
// changed the client identity.
func assertFourthLimited(t *testing.T, srv *Server, send func(i int) int) {
	t.Helper()
	codes := []int{send(1), send(2), send(3), send(4)}
	want := []int{200, 200, 200, 429}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("status codes = %v, want %v", codes, want)
		}
	}
}

func TestContact_ForwardedForFromUntrustedPeerIgnored(t *testing.T) {
	cfg := contactConfig(t)
	cfg.Server.TrustedProxies = []string{"192.0.2.0/24"}
	srv, _ := newContactServer(t, cfg)

	assertFourthLimited(t, srv, func(i int) int {
		return postContactFrom(srv, "198.51.100.9:1234", fmt.Sprintf("203.0.113.%d", i), "")
	})
}

func TestContact_ForwardedForChainStopsAtUntrustedHop(t *testing.T) {
	cfg := contactConfig(t)
	cfg.Server.TrustedProxies = []string{"192.0.2.0/24"}
	srv, _ := newContactServer(t, cfg)

	// 198.51.100.1 is not a trusted proxy, so it is the client; the entries
	// left of it are whatever that client chose to send.
	assertFourthLimited(t, srv, func(i int) int {
		return postContactFrom(srv, "192.0.2.1:1234", fmt.Sprintf("203.0.113.%d, 198.51.100.1", i), "")
	})
}

func TestContact_ForwardedForChainThroughTwoTrustedProxies(t *testing.T) {
	cfg := contactConfig(t)
	cfg.Server.TrustedProxies = []string{"192.0.2.0/24", "198.51.100.0/24"}
	srv, _ := newContactServer(t, cfg)

	for i := 1; i <= 4; i++ {
		xff := fmt.Sprintf("203.0.113.%d, 198.51.100.1", i)
		if code := postContactFrom(srv, "192.0.2.1:1234", xff, ""); code != http.StatusOK {
			t.Fatalf("client 203.0.113.%d: status = %d, want 200", i, code)
		}
	}
}

func TestContact_RealIPHeaderNotUsed(t *testing.T) {
	cfg := contactConfig(t)
	cfg.Server.TrustedProxies = []string{"192.0.2.0/24"}
	srv, _ := newContactServer(t, cfg)

	assertFourthLimited(t, srv, func(i int) int {
		return postContactFrom(srv, "192.0.2.1:1234", "", fmt.Sprintf("203.0.113.%d", i))
	})
}
