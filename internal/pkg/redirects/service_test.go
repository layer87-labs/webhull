package redirects

import (
	"strings"
	"testing"

	"github.com/layer87-labs/webhull/internal/pkg/config"
)

var testReserved = []string{"/static", "/api", "/health", "/sitemap.xml", "/robots.txt", "/gate", "/arcon", "/js/script.js"}

var testSlugs = []string{"", "start", "plattform", "leistungen", "produkte", "produkte/desk"}

func TestNewService_Normalizes(t *testing.T) {
	tests := []struct {
		name       string
		cfg        config.RedirectConfig
		wantFrom   string
		wantTo     string
		wantStatus int
	}{
		{
			name:       "defaults to 301",
			cfg:        config.RedirectConfig{From: "/souveraenitaet", To: "/plattform"},
			wantFrom:   "/souveraenitaet",
			wantTo:     "/plattform",
			wantStatus: 301,
		},
		{
			name:       "explicit 302",
			cfg:        config.RedirectConfig{From: "/zielgruppen", To: "/leistungen#zielgruppen", Status: 302},
			wantFrom:   "/zielgruppen",
			wantTo:     "/leistungen#zielgruppen",
			wantStatus: 302,
		},
		{
			name:       "leading slash added, trailing slash stripped",
			cfg:        config.RedirectConfig{From: "alt/", To: "neu"},
			wantFrom:   "/alt",
			wantTo:     "/neu",
			wantStatus: 301,
		},
		{
			name:       "surrounding whitespace trimmed",
			cfg:        config.RedirectConfig{From: "  /alt  ", To: "  /neu  ", Status: 308},
			wantFrom:   "/alt",
			wantTo:     "/neu",
			wantStatus: 308,
		},
		{
			name:       "absolute external URL",
			cfg:        config.RedirectConfig{From: "/blog", To: "https://blog.example.com/", Status: 307},
			wantFrom:   "/blog",
			wantTo:     "https://blog.example.com/",
			wantStatus: 307,
		},
		{
			name:       "nested from path",
			cfg:        config.RedirectConfig{From: "/produkte/alt", To: "/produkte/desk"},
			wantFrom:   "/produkte/alt",
			wantTo:     "/produkte/desk",
			wantStatus: 301,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewService([]config.RedirectConfig{tt.cfg}, testReserved, testSlugs)
			if err != nil {
				t.Fatalf("NewService failed: %v", err)
			}
			rules := svc.Rules()
			if len(rules) != 1 {
				t.Fatalf("got %d rules, want 1", len(rules))
			}
			if rules[0].From != tt.wantFrom {
				t.Errorf("From = %q, want %q", rules[0].From, tt.wantFrom)
			}
			if rules[0].To != tt.wantTo {
				t.Errorf("To = %q, want %q", rules[0].To, tt.wantTo)
			}
			if rules[0].Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", rules[0].Status, tt.wantStatus)
			}
		})
	}
}

func TestNewService_Rejects(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.RedirectConfig
		wantErr string
	}{
		{"empty from", config.RedirectConfig{From: "", To: "/x"}, "must not be empty"},
		{"empty to", config.RedirectConfig{From: "/a", To: ""}, "must not be empty"},
		{"root from", config.RedirectConfig{From: "/", To: "/x"}, "site root"},
		{"whitespace in from", config.RedirectConfig{From: "/a b", To: "/x"}, "whitespace"},
		{"query in from", config.RedirectConfig{From: "/a?x=1", To: "/x"}, "query string"},
		{"fragment in from", config.RedirectConfig{From: "/a#x", To: "/x"}, "fragment"},
		{"url as from", config.RedirectConfig{From: "https://example.com/a", To: "/x"}, "not a URL"},
		{"empty segment in from", config.RedirectConfig{From: "/a//b", To: "/x"}, "empty path segments"},
		{"dotdot in from", config.RedirectConfig{From: "/a/../b", To: "/x"}, `".."`},
		{"dot in from", config.RedirectConfig{From: "/a/./b", To: "/x"}, `"."`},
		{"whitespace in to", config.RedirectConfig{From: "/a", To: "/x y"}, "whitespace"},
		{"protocol-relative to", config.RedirectConfig{From: "/a", To: "//evil.example.com"}, "protocol-relative"},
		{"bare fragment to", config.RedirectConfig{From: "/a", To: "#top"}, "bare fragment"},
		{"javascript scheme", config.RedirectConfig{From: "/a", To: "javascript://x"}, "http(s)"},
		{"ftp scheme", config.RedirectConfig{From: "/a", To: "ftp://files.example.com/"}, "http(s)"},
		{"status 200", config.RedirectConfig{From: "/a", To: "/x", Status: 200}, "status 200"},
		{"status 303", config.RedirectConfig{From: "/a", To: "/x", Status: 303}, "status 303"},
		{"self redirect", config.RedirectConfig{From: "/a", To: "/a"}, "itself"},
		{"self redirect with fragment", config.RedirectConfig{From: "/a", To: "/a#x"}, "itself"},
		{"self redirect trailing slash", config.RedirectConfig{From: "/a/", To: "/a"}, "itself"},
		{"reserved exact", config.RedirectConfig{From: "/health", To: "/x"}, "built-in route"},
		{"reserved prefix", config.RedirectConfig{From: "/static/old.css", To: "/x"}, "built-in route"},
		{"reserved api", config.RedirectConfig{From: "/api/contact", To: "/x"}, "built-in route"},
		{"reserved arcon", config.RedirectConfig{From: "/arcon", To: "/x"}, "built-in route"},
		{"reserved plausible script", config.RedirectConfig{From: "/js/script.js", To: "/x"}, "built-in route"},
		{"slug collision", config.RedirectConfig{From: "/plattform", To: "/x"}, "existing page slug"},
		{"slug collision trailing slash", config.RedirectConfig{From: "/plattform/", To: "/x"}, "existing page slug"},
		{"slug collision missing slash", config.RedirectConfig{From: "plattform", To: "/x"}, "existing page slug"},
		{"nested slug collision", config.RedirectConfig{From: "/produkte/desk", To: "/x"}, "existing page slug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService([]config.RedirectConfig{tt.cfg}, testReserved, testSlugs)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewService_ReservedPrefixDoesNotMatchSiblings(t *testing.T) {
	// "/static" is reserved, "/statistik" is not — the prefix check must
	// respect path-segment boundaries.
	svc, err := NewService([]config.RedirectConfig{
		{From: "/statistik", To: "/x"},
		{From: "/gateway", To: "/x"},
	}, testReserved, testSlugs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.Rules()) != 2 {
		t.Errorf("got %d rules, want 2", len(svc.Rules()))
	}
}

func TestNewService_DuplicateFrom(t *testing.T) {
	_, err := NewService([]config.RedirectConfig{
		{From: "/alt", To: "/neu"},
		{From: "alt/", To: "/anders"},
	}, testReserved, testSlugs)
	if err == nil {
		t.Fatal("expected error for duplicate from, got nil")
	}
	if !strings.Contains(err.Error(), "redirects[1]") || !strings.Contains(err.Error(), "already declared by redirects[0]") {
		t.Errorf("error = %q, want it to name both entries", err.Error())
	}
}

func TestNewService_ErrorNamesIndex(t *testing.T) {
	_, err := NewService([]config.RedirectConfig{
		{From: "/ok", To: "/x"},
		{From: "/health", To: "/x"},
	}, testReserved, testSlugs)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "redirects[1]:") {
		t.Errorf("error = %q, want prefix \"redirects[1]:\"", err.Error())
	}
}

func TestNewService_Empty(t *testing.T) {
	svc, err := NewService(nil, testReserved, testSlugs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.Rules()) != 0 {
		t.Errorf("got %d rules, want 0", len(svc.Rules()))
	}
}

func TestNewService_PreservesOrder(t *testing.T) {
	svc, err := NewService([]config.RedirectConfig{
		{From: "/c", To: "/x"},
		{From: "/a", To: "/x"},
		{From: "/b", To: "/x"},
	}, testReserved, testSlugs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := []string{svc.Rules()[0].From, svc.Rules()[1].From, svc.Rules()[2].From}
	want := []string{"/c", "/a", "/b"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rules[%d].From = %q, want %q", i, got[i], want[i])
		}
	}
}
