package redirects

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/layer87-labs/webhull/internal/pkg/config"
)

// DefaultStatus is used when a redirect declares no status code.
const DefaultStatus = http.StatusMovedPermanently

// allowedStatus lists the status codes a redirect may use. 301/308 are
// permanent, 302/307 temporary; 307/308 preserve the request method.
var allowedStatus = map[int]bool{
	http.StatusMovedPermanently:  true, // 301
	http.StatusFound:             true, // 302
	http.StatusTemporaryRedirect: true, // 307
	http.StatusPermanentRedirect: true, // 308
}

// Service holds the validated redirect rules of a site.
type Service struct {
	rules []Rule
}

// NewService normalizes and validates the configured redirects.
//
// reserved lists the paths owned by built-in routes ("/static", "/api", …);
// a redirect may neither equal one nor live beneath it. slugs lists every
// registered page slug across all languages — a redirect that shadows a page
// would make that page unreachable, so it is rejected at startup.
func NewService(cfgs []config.RedirectConfig, reserved []string, slugs []string) (*Service, error) {
	slugPaths := make(map[string]bool, len(slugs))
	for _, slug := range slugs {
		slugPaths["/"+slug] = true
	}

	seen := make(map[string]int, len(cfgs))
	rules := make([]Rule, 0, len(cfgs))

	for i, cfg := range cfgs {
		from, err := normalizeFrom(cfg.From)
		if err != nil {
			return nil, fmt.Errorf("redirects[%d]: from %q: %w", i, cfg.From, err)
		}

		to, err := normalizeTo(cfg.To)
		if err != nil {
			return nil, fmt.Errorf("redirects[%d]: to %q: %w", i, cfg.To, err)
		}

		status := cfg.Status
		if status == 0 {
			status = DefaultStatus
		}
		if !allowedStatus[status] {
			return nil, fmt.Errorf("redirects[%d]: status %d is not allowed (use 301, 302, 307 or 308)", i, status)
		}

		if pathOf(to) == from {
			return nil, fmt.Errorf("redirects[%d]: %q redirects to itself", i, from)
		}

		for _, r := range reserved {
			if from == r || strings.HasPrefix(from, r+"/") {
				return nil, fmt.Errorf("redirects[%d]: from %q collides with the built-in route %q", i, from, r)
			}
		}

		if slugPaths[from] {
			return nil, fmt.Errorf("redirects[%d]: from %q collides with an existing page slug — remove the page or the redirect", i, from)
		}

		if prev, dup := seen[from]; dup {
			return nil, fmt.Errorf("redirects[%d]: from %q is already declared by redirects[%d]", i, from, prev)
		}
		seen[from] = i

		rules = append(rules, Rule{From: from, To: to, Status: status})
	}

	return &Service{rules: rules}, nil
}

// Rules returns the validated rules in declaration order.
func (s *Service) Rules() []Rule {
	return s.rules
}

// normalizeFrom turns a configured source path into its canonical form:
// leading slash guaranteed, trailing slash stripped, no query or fragment,
// no empty or dot segments.
func normalizeFrom(raw string) (string, error) {
	from := strings.TrimSpace(raw)
	if from == "" {
		return "", fmt.Errorf("must not be empty")
	}
	if strings.ContainsAny(from, " \t\r\n") {
		return "", fmt.Errorf("must not contain whitespace")
	}
	if strings.ContainsAny(from, "?#") {
		return "", fmt.Errorf("must be a plain path without query string or fragment")
	}
	if strings.Contains(from, "://") {
		return "", fmt.Errorf("must be a path, not a URL")
	}

	from = "/" + strings.Trim(from, "/")
	if from == "/" {
		return "", fmt.Errorf("must not be the site root")
	}

	for _, seg := range strings.Split(from[1:], "/") {
		switch seg {
		case "":
			return "", fmt.Errorf("must not contain empty path segments")
		case ".", "..":
			return "", fmt.Errorf("must not contain %q segments", seg)
		}
	}

	return from, nil
}

// normalizeTo validates the target: either an absolute http(s) URL or an
// internal path, which gets a leading slash when missing.
func normalizeTo(raw string) (string, error) {
	to := strings.TrimSpace(raw)
	if to == "" {
		return "", fmt.Errorf("must not be empty")
	}
	if strings.ContainsAny(to, " \t\r\n") {
		return "", fmt.Errorf("must not contain whitespace")
	}

	if strings.Contains(to, "://") {
		u, err := url.Parse(to)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "", fmt.Errorf("external targets must be absolute http(s) URLs")
		}
		return to, nil
	}

	if strings.HasPrefix(to, "//") {
		return "", fmt.Errorf("protocol-relative targets are not allowed")
	}
	if strings.HasPrefix(to, "#") {
		return "", fmt.Errorf("must be a path — a bare fragment has nothing to attach to")
	}
	if !strings.HasPrefix(to, "/") {
		to = "/" + to
	}

	return to, nil
}

// pathOf strips query string and fragment from an internal target so a
// self-redirect ("/a" → "/a#x") is caught.
func pathOf(to string) string {
	if idx := strings.IndexAny(to, "?#"); idx >= 0 {
		to = to[:idx]
	}
	return strings.TrimRight(to, "/")
}
