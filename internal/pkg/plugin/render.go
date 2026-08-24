package plugin

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// renderData is passed to the render template.
type renderData struct {
	Items []Item
}

// funcMap provides small, generic helpers for render templates. Kept
// deliberately minimal — plugin templates are HTML authored by whoever
// wires up the plugin, not application code.
var funcMap = template.FuncMap{
	"hasSuffix": strings.HasSuffix,
	"contains":  strings.Contains,
	"default": func(fallback, v interface{}) interface{} {
		if v == nil || v == "" {
			return fallback
		}
		return v
	},
	"index0": func(items []Item, i int) interface{} {
		if i < 0 || i >= len(items) {
			return nil
		}
		return items[i]
	},
}

// loadTemplate parses the manifest's render template (relative to the
// plugin's own directory). html/template auto-escapes every interpolated
// value, so allowlisted-but-untrusted upstream data (item titles, labels,
// image URLs) can never break out of the surrounding markup.
func loadTemplate(m *Manifest) (*template.Template, error) {
	path := filepath.Join(m.dir, m.Render.Template)
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read render template %s: %w", path, err)
	}
	tmpl, err := template.New(m.Render.Template).Funcs(funcMap).Parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse render template %s: %w", path, err)
	}
	return tmpl, nil
}

// renderHTML executes the template against the allowlisted items and
// returns the resulting HTML fragment.
func renderHTML(tmpl *template.Template, items []Item) (string, error) {
	var b strings.Builder
	if err := tmpl.Execute(&b, renderData{Items: items}); err != nil {
		return "", fmt.Errorf("execute render template: %w", err)
	}
	return b.String(), nil
}
