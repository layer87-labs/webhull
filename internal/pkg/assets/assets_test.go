package assets

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestNewService_HashesFiles(t *testing.T) {
	dir := t.TempDir()
	cssDir := filepath.Join(dir, "css")
	jsDir := filepath.Join(dir, "js")
	os.MkdirAll(cssDir, 0o755)
	os.MkdirAll(jsDir, 0o755)

	os.WriteFile(filepath.Join(cssDir, "style.css"), []byte("body { color: red; }"), 0o644)
	os.WriteFile(filepath.Join(jsDir, "app.js"), []byte("console.log('hello')"), 0o644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not hashed"), 0o644)

	logger := zap.NewNop()
	svc := NewService(dir, "/static", logger)

	cssPath := svc.Path("/static/css/style.css")
	if cssPath == "/static/css/style.css" {
		t.Error("expected CSS path to have hash query param")
	}
	if len(svc.Hash("/static/css/style.css")) != 8 {
		t.Errorf("expected 8-char hash, got %q", svc.Hash("/static/css/style.css"))
	}

	jsPath := svc.Path("/static/js/app.js")
	if jsPath == "/static/js/app.js" {
		t.Error("expected JS path to have hash query param")
	}

	txtPath := svc.Path("/static/readme.txt")
	if txtPath != "/static/readme.txt" {
		t.Errorf("expected TXT path unchanged, got %q", txtPath)
	}
}

func TestPath_UnknownAsset(t *testing.T) {
	logger := zap.NewNop()
	svc := NewService("", "/static", logger)

	result := svc.Path("/static/unknown.css")
	if result != "/static/unknown.css" {
		t.Errorf("expected unchanged path for unknown asset, got %q", result)
	}
}

func TestHash_Deterministic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.css"), []byte("body{}"), 0o644)

	logger := zap.NewNop()
	svc1 := NewService(dir, "/static", logger)
	svc2 := NewService(dir, "/static", logger)

	hash1 := svc1.Hash("/static/test.css")
	hash2 := svc2.Hash("/static/test.css")

	if hash1 != hash2 {
		t.Errorf("hashes should be deterministic: %q vs %q", hash1, hash2)
	}
}

func TestHash_ChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.js")

	os.WriteFile(file, []byte("v1"), 0o644)
	logger := zap.NewNop()
	svc1 := NewService(dir, "/static", logger)
	hash1 := svc1.Hash("/static/app.js")

	os.WriteFile(file, []byte("v2"), 0o644)
	svc2 := NewService(dir, "/static", logger)
	hash2 := svc2.Hash("/static/app.js")

	if hash1 == hash2 {
		t.Error("hash should change when file content changes")
	}
}
