package plugin

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func writePluginDir(t *testing.T, dir, name, manifestYAML, templateHTML string) {
	t.Helper()
	pluginDir := filepath.Join(dir, name)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifestYAML), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "fragment.tmpl.html"), []byte(templateHTML), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
}

func TestService_EndToEnd(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"articles": [
			{"id": 1, "make": "CARADO"},
			{"id": 2, "make": "KNAUS"}
		]}`))
	}))
	defer upstream.Close()

	dir := t.TempDir()
	manifest := `
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: e2e
source:
  url: ` + upstream.URL + `/articles
  refreshInterval: 30s
select:
  root: articles
  fields: [id, make]
render:
  template: fragment.tmpl.html
  into:
    page: vermietung
    contentKey: fleet
csp:
  imgSrc: ["https://cdn.example.com"]
`
	template := `{{range .Items}}<div>{{.make}}</div>{{end}}`
	writePluginDir(t, dir, "e2e", manifest, template)

	svc, err := NewService(dir, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Stop()

	content := svc.ContentFor("vermietung")
	html, ok := content["fleet"]
	if !ok {
		t.Fatal("expected content for page=vermietung contentKey=fleet")
	}
	if !strings.Contains(html, "CARADO") || !strings.Contains(html, "KNAUS") {
		t.Errorf("rendered html missing expected items: %q", html)
	}

	if got := svc.ContentFor("other-page"); got != nil {
		t.Errorf("expected nil for page with no plugins, got %v", got)
	}

	hosts := svc.ImgSrcHosts()
	if len(hosts) != 1 || hosts[0] != "https://cdn.example.com" {
		t.Errorf("ImgSrcHosts = %v, want [https://cdn.example.com]", hosts)
	}
}

func TestService_DuplicateNameRejected(t *testing.T) {
	dir := t.TempDir()
	manifest := strings.ReplaceAll(validManifest, "into:\n    page: vermietung\n    contentKey: fleet",
		"into:\n    page: vermietung\n    contentKey: fleetA")
	writePluginDir(t, dir, "a", manifest, "<div></div>")
	writePluginDir(t, dir, "b", manifest, "<div></div>") // same name "testsource"

	_, err := NewService(dir, zap.NewNop())
	if err == nil {
		t.Fatal("expected error for duplicate plugin name")
	}
}

func TestService_DuplicateRenderTargetRejected(t *testing.T) {
	dir := t.TempDir()
	manifestA := strings.ReplaceAll(validManifest, "name: testsource", "name: sourceA")
	manifestB := strings.ReplaceAll(validManifest, "name: testsource", "name: sourceB")
	writePluginDir(t, dir, "a", manifestA, "<div></div>")
	writePluginDir(t, dir, "b", manifestB, "<div></div>") // same into.page/contentKey

	_, err := NewService(dir, zap.NewNop())
	if err == nil {
		t.Fatal("expected error for two plugins claiming the same page/contentKey")
	}
}

func TestInstance_StaleWhileError(t *testing.T) {
	m := &Manifest{
		Name: "stale-test",
		Source: Source{
			Timeout:         time.Second,
			StaleWhileError: 50 * time.Millisecond,
		},
	}
	tmpl, err := template.New("inline").Parse("<div>{{len .Items}}</div>")
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	in := newInstance(m, tmpl, zap.NewNop())

	in.mu.Lock()
	in.html = "<div>cached</div>"
	in.lastGood = time.Now()
	in.haveGood = true
	in.mu.Unlock()

	if got := in.content(); got != "<div>cached</div>" {
		t.Fatalf("content() = %q, want cached fragment while fresh", got)
	}

	time.Sleep(60 * time.Millisecond)

	if got := in.content(); got != "" {
		t.Fatalf("content() = %q, want empty after StaleWhileError elapsed", got)
	}
}
