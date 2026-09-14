package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/pkg/config"
)

// writeContent writes a minimal content page with frontmatter.
func writeContent(t *testing.T, dir, lang, file, frontmatter string) {
	t.Helper()
	path := filepath.Join(dir, lang, file)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\n" + frontmatter + "---\n<p>" + file + "</p>\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// testSiteConfig builds a two-language site with a nested page and a set of
// redirects. It mirrors the shape a consumer repo's pages.yaml produces after
// config.Load and content.Load.
func testSiteConfig(t *testing.T, redirects []config.RedirectConfig) *config.SiteConfig {
	t.Helper()
	contentDir := t.TempDir()

	writeContent(t, contentDir, "de", "start.html", "id: home\ntemplate: home\ntitle: Start\ndescription: DE\n")
	writeContent(t, contentDir, "en", "home.html", "id: home\ntemplate: home\ntitle: Home\ndescription: EN\n")
	writeContent(t, contentDir, "de", "produkte.html", "id: products\ntemplate: default\ntitle: Produkte\ndescription: DE\n")
	writeContent(t, contentDir, "en", "products.html", "id: products\ntemplate: default\ntitle: Products\ndescription: EN\n")
	writeContent(t, contentDir, "de", "produkte-desk.html", "id: product-desk\ntemplate: default\ntitle: Desk\ndescription: DE\nslug: produkte/desk\n")
	writeContent(t, contentDir, "en", "products-desk.html", "id: product-desk\ntemplate: default\ntitle: Desk\ndescription: EN\nslug: products/desk\n")

	disabled := false
	return &config.SiteConfig{
		Site: config.SiteIdentity{Name: "Test", BaseURL: "https://example.com"},
		I18n: config.I18nConfig{DefaultLanguage: "de", Languages: []string{"de", "en"}},
		Navigation: config.NavigationConfig{
			Header: map[string][]config.NavItemConfig{
				"de": {{Slug: "produkte", Title: "Produkte", URL: "/produkte"}},
				"en": {{Slug: "products", Title: "Products", URL: "/products"}},
			},
		},
		ContentDir: contentDir,
		Redirects:  redirects,
		Server:     config.ServerConfig{Environment: "production"},
		Health:     config.HealthConfig{Enabled: &disabled},
	}
}

func newTestServer(t *testing.T, redirects []config.RedirectConfig) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	srv, err := New(testSiteConfig(t, redirects), zap.NewNop())
	if err != nil {
		t.Fatalf("server.New failed: %v", err)
	}
	return srv
}

func do(srv *Server, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	srv.Router().ServeHTTP(rec, req)
	return rec
}

func TestRedirects_StatusAndLocation(t *testing.T) {
	srv := newTestServer(t, []config.RedirectConfig{
		{From: "/souveraenitaet", To: "/plattform"},
		{From: "/zielgruppen", To: "/leistungen#zielgruppen", Status: 302},
		{From: "blog", To: "https://blog.example.com/", Status: 308},
	})

	tests := []struct {
		path         string
		wantStatus   int
		wantLocation string
	}{
		{"/souveraenitaet", 301, "/plattform"},
		{"/souveraenitaet/", 301, "/plattform"}, // trailing-slash spelling answers directly
		{"/zielgruppen", 302, "/leistungen#zielgruppen"},
		{"/zielgruppen/", 302, "/leistungen#zielgruppen"},
		{"/blog", 308, "https://blog.example.com/"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				rec := do(srv, method, tt.path)
				if rec.Code != tt.wantStatus {
					t.Errorf("%s %s: status = %d, want %d", method, tt.path, rec.Code, tt.wantStatus)
				}
				if loc := rec.Header().Get("Location"); loc != tt.wantLocation {
					t.Errorf("%s %s: Location = %q, want %q", method, tt.path, loc, tt.wantLocation)
				}
			}
		})
	}
}

func TestRedirects_DoNotAffectOtherRoutes(t *testing.T) {
	srv := newTestServer(t, []config.RedirectConfig{
		{From: "/souveraenitaet", To: "/plattform"},
	})

	if rec := do(srv, http.MethodGet, "/gibt-es-nicht"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown path: status = %d, want 404", rec.Code)
	}
	if rec := do(srv, http.MethodGet, "/souveraenitaet/unterseite"); rec.Code != http.StatusNotFound {
		t.Errorf("path beneath a redirect: status = %d, want 404", rec.Code)
	}
	if rec := do(srv, http.MethodGet, "/produkte"); rec.Code != http.StatusOK {
		t.Errorf("/produkte: status = %d, want 200", rec.Code)
	}
}

func TestRedirects_NotInSitemap(t *testing.T) {
	srv := newTestServer(t, []config.RedirectConfig{
		{From: "/souveraenitaet", To: "/plattform"},
	})

	rec := do(srv, http.MethodGet, "/sitemap.xml")
	if rec.Code != http.StatusOK {
		t.Fatalf("sitemap: status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "souveraenitaet") {
		t.Error("sitemap must not list redirect sources")
	}
}

func TestRedirects_SlugCollisionFailsStartup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testSiteConfig(t, []config.RedirectConfig{
		{From: "/produkte/desk", To: "/produkte"},
	})
	_, err := New(cfg, zap.NewNop())
	if err == nil {
		t.Fatal("expected startup error for redirect colliding with a page slug, got nil")
	}
	if !strings.Contains(err.Error(), "existing page slug") {
		t.Errorf("error = %q, want it to explain the slug collision", err.Error())
	}
}

func TestRedirects_ReservedRouteFailsStartup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testSiteConfig(t, []config.RedirectConfig{
		{From: "/api/legacy", To: "/produkte"},
	})
	_, err := New(cfg, zap.NewNop())
	if err == nil {
		t.Fatal("expected startup error for redirect beneath a built-in route, got nil")
	}
	if !strings.Contains(err.Error(), "built-in route") {
		t.Errorf("error = %q, want it to name the built-in route", err.Error())
	}
}

func TestNestedSlug_RouteAndSitemap(t *testing.T) {
	srv := newTestServer(t, nil)

	// Parent and nested page coexist without a router conflict.
	for _, path := range []string{"/produkte", "/produkte/desk", "/products/desk"} {
		rec := do(srv, http.MethodGet, path)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", path, rec.Code)
		}
	}

	// Trailing slash on the nested route is redirected to the canonical path.
	rec := do(srv, http.MethodGet, "/produkte/desk/")
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/produkte/desk" {
		t.Errorf("/produkte/desk/: got %d %q, want 301 to /produkte/desk", rec.Code, rec.Header().Get("Location"))
	}

	// Canonical, hreflang and the nav active state are derived from the slug.
	rec = do(srv, http.MethodGet, "/produkte/desk")
	body := rec.Body.String()
	for _, want := range []string{
		`https://example.com/produkte/desk`,
		`hreflang="en"`,
		`https://example.com/products/desk`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/produkte/desk body should contain %q", want)
		}
	}

	rec = do(srv, http.MethodGet, "/sitemap.xml")
	sitemap := rec.Body.String()
	for _, want := range []string{
		"<loc>https://example.com/produkte/desk</loc>",
		"<loc>https://example.com/products/desk</loc>",
		`hreflang="en" href="https://example.com/products/desk"`,
	} {
		if !strings.Contains(sitemap, want) {
			t.Errorf("sitemap should contain %q", want)
		}
	}
}

func TestNestedSlug_NavActiveState(t *testing.T) {
	srv := newTestServer(t, nil)

	header := srv.Navigation.ResolveHeader("de", srv.Pages.Resolve("produkte/desk").Slug)
	if len(header.Items) != 1 || !header.Items[0].Active {
		t.Errorf("header item /produkte should be active on /produkte/desk, got %+v", header.Items)
	}
}
