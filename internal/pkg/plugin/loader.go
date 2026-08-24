package plugin

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const manifestFile = "plugin.yaml"

const supportedAPIVersion = "webhull.layer87.de/v1"

// secretRefPattern matches a header value that is *entirely* an env
// reference: "${VAR}" or "${VAR:default}". Anything else (a literal string,
// or a literal with an interpolated suffix) is rejected — headers are the
// one place credentials flow through a manifest, so partial matches are not
// good enough.
var secretRefPattern = regexp.MustCompile(`^\$\{[A-Za-z_][A-Za-z0-9_]*(:[^}]*)?\}$`)

// discover finds every plugin.yaml in immediate subdirectories of dir.
// Missing dir is not an error — plugins are opt-in.
func discover(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read plugins dir %s: %w", dir, err)
	}

	var paths []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name(), manifestFile)
		if _, err := os.Stat(p); err == nil {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths) // deterministic load order
	return paths, nil
}

// loadManifest reads, validates and env-expands a single plugin.yaml.
//
// It parses the file twice: once on the raw bytes (to validate that header
// values are pure ${VAR} references, before any expansion could hide a
// literal secret), and once on the env-expanded bytes (the manifest actually
// used at runtime).
func loadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var rawManifest Manifest
	if err := yaml.Unmarshal(raw, &rawManifest); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for key, val := range rawManifest.Source.Headers {
		if !secretRefPattern.MatchString(val) {
			return nil, fmt.Errorf(
				"%s: source.headers[%q] must be exactly \"${VAR}\" or \"${VAR:default}\" — "+
					"literal header values are not allowed (would commit a secret)", path, key)
		}
	}

	expanded := expandEnvSafe(string(raw))
	var m Manifest
	if err := yaml.Unmarshal([]byte(expanded), &m); err != nil {
		return nil, fmt.Errorf("parse %s (expanded): %w", path, err)
	}
	m.dir = filepath.Dir(path)

	if err := validateManifest(&m, path); err != nil {
		return nil, err
	}
	applyManifestDefaults(&m)
	return &m, nil
}

func validateManifest(m *Manifest, path string) error {
	if m.APIVersion != supportedAPIVersion {
		return fmt.Errorf("%s: apiVersion must be %q, got %q", path, supportedAPIVersion, m.APIVersion)
	}
	if m.Kind != "HTTPDataSource" {
		return fmt.Errorf("%s: unsupported kind %q (only \"HTTPDataSource\" is implemented)", path, m.Kind)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%s: name is required", path)
	}
	if m.Source.Type != "" && m.Source.Type != "http" {
		return fmt.Errorf("%s: source.type %q is not supported (only \"http\")", path, m.Source.Type)
	}
	if m.Source.URL == "" {
		return fmt.Errorf("%s: source.url is required", path)
	}
	u, err := url.Parse(m.Source.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%s: source.url must be an absolute http(s) URL, got %q", path, m.Source.URL)
	}
	if len(m.Select.Fields) == 0 {
		return fmt.Errorf("%s: select.fields must list at least one field — "+
			"an empty allowlist would mean the plugin has nothing to render", path)
	}
	if m.Render.Template == "" {
		return fmt.Errorf("%s: render.template is required", path)
	}
	if _, err := os.Stat(filepath.Join(m.dir, m.Render.Template)); err != nil {
		return fmt.Errorf("%s: render.template %q not found next to plugin.yaml", path, m.Render.Template)
	}
	if m.Render.Into.Page == "" || m.Render.Into.ContentKey == "" {
		return fmt.Errorf("%s: render.into.page and render.into.contentKey are both required", path)
	}
	for _, host := range m.CSP.ImgSrc {
		if host == "*" || strings.Contains(host, "*") {
			return fmt.Errorf("%s: csp.imgSrc entries must be exact hosts, not wildcards (%q)", path, host)
		}
		if hu, err := url.Parse(host); err != nil || hu.Scheme == "" || hu.Host == "" {
			return fmt.Errorf("%s: csp.imgSrc entry %q must be a full origin, e.g. \"https://cdn.example.com\"", path, host)
		}
	}
	return nil
}

func applyManifestDefaults(m *Manifest) {
	if m.Source.Type == "" {
		m.Source.Type = "http"
	}
	if m.Source.Timeout <= 0 {
		m.Source.Timeout = 8 * time.Second
	}
	if m.Source.RefreshInterval <= 0 {
		m.Source.RefreshInterval = 15 * time.Minute
	} else if m.Source.RefreshInterval < 30*time.Second {
		m.Source.RefreshInterval = 30 * time.Second
	}
	if m.Source.StaleWhileError <= 0 {
		m.Source.StaleWhileError = 24 * time.Hour
	}
}

// expandEnvSafe expands ${VAR} and ${VAR:default} against the process
// environment. Mirrors config.expandEnvSafe; duplicated rather than
// imported because pkg/ packages must not depend on each other
// (see AGENTS.md — "no cross-domain imports between pkg/ packages").
func expandEnvSafe(s string) string {
	return os.Expand(s, func(key string) string {
		if idx := strings.Index(key, ":"); idx >= 0 {
			envKey := key[:idx]
			defaultVal := key[idx+1:]
			if v := os.Getenv(envKey); v != "" {
				return v
			}
			return defaultVal
		}
		return os.Getenv(key)
	})
}
