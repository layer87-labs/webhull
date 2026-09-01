// Package staticassets embeds webhull's built-in static JS files into the binary.
// These files are served at /static/js/* and can be overridden by placing a file
// with the same name in the user-configured staticDir.
package staticassets

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// FS holds the embedded static assets shipped with webhull.
// The directory structure mirrors the URL path under /static:
//
//	js/theme-init.js  →  /static/js/theme-init.js
//	js/menu.js        →  /static/js/menu.js
//	js/consent.js     →  /static/js/consent.js
//	js/contact.js     →  /static/js/contact.js
//	js/analytics.js   →  /static/js/analytics.js
//	js/errortracking-init.js → /static/js/errortracking-init.js
//
//go:embed js css
var FS embed.FS

// WrapFS returns an http.FileSystem backed by the embedded FS, rooted at the
// top level (so /js/menu.js is accessible as http.FileSystem.Open("/js/menu.js")).
func WrapFS() fs.FS {
	return FS
}

// overlayFS is an http.FileSystem that tries the user's staticDir first and
// falls back to the embedded built-ins. This ensures project files always take
// precedence over webhull's defaults.
type overlayFS struct {
	userDir  string
	fallback http.FileSystem
}

// OverlayFS returns an http.FileSystem that serves files from userDir first,
// falling back to the embedded built-in assets for any file not found there.
func OverlayFS(userDir string, fallback http.FileSystem) http.FileSystem {
	return overlayFS{userDir: userDir, fallback: fallback}
}

func (o overlayFS) Open(name string) (http.File, error) {
	// Try user dir first.
	if o.userDir != "" {
		p := filepath.Join(o.userDir, filepath.FromSlash(name))
		f, err := os.Open(p)
		if err == nil {
			return f, nil
		}
	}
	// Fall back to embedded assets.
	return o.fallback.Open(name)
}
