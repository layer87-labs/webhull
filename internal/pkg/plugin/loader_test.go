package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, dir, name, yaml string) {
	t.Helper()
	pluginDir := filepath.Join(dir, name)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "fragment.tmpl.html"), []byte("<div>{{len .Items}}</div>"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
}

const validManifest = `
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: testsource
source:
  url: https://api.example.com/v1/items
  query:
    locale: de-DE
  headers:
    Authorization: "${TEST_TOKEN}"
select:
  root: items
  fields:
    - id
    - name
render:
  template: fragment.tmpl.html
  into:
    page: vermietung
    contentKey: fleet
csp:
  imgSrc:
    - https://cdn.example.com
`

func TestLoadManifest_Valid(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "test", validManifest)

	t.Setenv("TEST_TOKEN", "secret-value")

	m, err := loadManifest(filepath.Join(dir, "test", "plugin.yaml"))
	if err != nil {
		t.Fatalf("loadManifest: %v", err)
	}
	if m.Name != "testsource" {
		t.Errorf("Name = %q, want testsource", m.Name)
	}
	if m.Source.Headers["Authorization"] != "secret-value" {
		t.Errorf("header not expanded, got %q", m.Source.Headers["Authorization"])
	}
	if m.Source.Type != "http" {
		t.Errorf("Type default = %q, want http", m.Source.Type)
	}
	if m.Source.Timeout <= 0 {
		t.Error("Timeout default not applied")
	}
}

func TestLoadManifest_LiteralSecretRejected(t *testing.T) {
	dir := t.TempDir()
	bad := `
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: leaky
source:
  url: https://api.example.com/v1/items
  headers:
    Authorization: "Bearer sk-literal-secret-123"
select:
  fields: [id]
render:
  template: fragment.tmpl.html
  into: { page: vermietung, contentKey: fleet }
`
	writeManifest(t, dir, "leaky", bad)

	_, err := loadManifest(filepath.Join(dir, "leaky", "plugin.yaml"))
	if err == nil {
		t.Fatal("expected error for literal secret in header, got nil")
	}
}

func TestLoadManifest_PartialEnvRefRejected(t *testing.T) {
	dir := t.TempDir()
	bad := `
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: leaky2
source:
  url: https://api.example.com/v1/items
  headers:
    Authorization: "Bearer ${TOKEN}"
select:
  fields: [id]
render:
  template: fragment.tmpl.html
  into: { page: vermietung, contentKey: fleet }
`
	writeManifest(t, dir, "leaky2", bad)

	_, err := loadManifest(filepath.Join(dir, "leaky2", "plugin.yaml"))
	if err == nil {
		t.Fatal("expected error: header value is not purely a ${VAR} reference")
	}
}

func TestLoadManifest_EmptyFieldsRejected(t *testing.T) {
	dir := t.TempDir()
	bad := `
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: nofields
source:
  url: https://api.example.com/v1/items
select: {}
render:
  template: fragment.tmpl.html
  into: { page: vermietung, contentKey: fleet }
`
	writeManifest(t, dir, "nofields", bad)

	_, err := loadManifest(filepath.Join(dir, "nofields", "plugin.yaml"))
	if err == nil {
		t.Fatal("expected error for empty select.fields (deny by default)")
	}
}

func TestLoadManifest_WrongAPIVersionRejected(t *testing.T) {
	dir := t.TempDir()
	bad := `
apiVersion: v2
kind: HTTPDataSource
name: wrongver
source: { url: https://api.example.com/v1/items }
select: { fields: [id] }
render:
  template: fragment.tmpl.html
  into: { page: vermietung, contentKey: fleet }
`
	writeManifest(t, dir, "wrongver", bad)

	_, err := loadManifest(filepath.Join(dir, "wrongver", "plugin.yaml"))
	if err == nil {
		t.Fatal("expected error for unsupported apiVersion")
	}
}

func TestLoadManifest_WildcardCSPRejected(t *testing.T) {
	dir := t.TempDir()
	bad := `
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: wildcsp
source: { url: https://api.example.com/v1/items }
select: { fields: [id] }
render:
  template: fragment.tmpl.html
  into: { page: vermietung, contentKey: fleet }
csp:
  imgSrc: ["https://*.example.com"]
`
	writeManifest(t, dir, "wildcsp", bad)

	_, err := loadManifest(filepath.Join(dir, "wildcsp", "plugin.yaml"))
	if err == nil {
		t.Fatal("expected error for wildcard csp.imgSrc")
	}
}

func TestLoadManifest_MissingTemplateRejected(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "notmpl")
	os.MkdirAll(pluginDir, 0o755)
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`
apiVersion: webhull.layer87.de/v1
kind: HTTPDataSource
name: notmpl
source: { url: https://api.example.com/v1/items }
select: { fields: [id] }
render:
  template: does-not-exist.tmpl.html
  into: { page: vermietung, contentKey: fleet }
`), 0o644)

	_, err := loadManifest(filepath.Join(pluginDir, "plugin.yaml"))
	if err == nil {
		t.Fatal("expected error for missing render.template file")
	}
}

func TestDiscover_MissingDirIsNotError(t *testing.T) {
	paths, err := discover(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("discover on missing dir: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected no plugins, got %v", paths)
	}
}

func TestDiscover_FindsManifests(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "a", validManifest)
	writeManifest(t, dir, "b", validManifest)
	os.MkdirAll(filepath.Join(dir, "not-a-plugin"), 0o755) // dir without plugin.yaml

	paths, err := discover(dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 manifests, got %d: %v", len(paths), paths)
	}
}
