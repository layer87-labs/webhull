package plugin

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Service loads and runs every plugin found in a directory. It is the only
// exported type consumed outside this package.
type Service struct {
	instances []*instance
	// byPage maps page ID → contentKey → *instance, for O(1) lookup from
	// buildPageData on every request.
	byPage map[string]map[string]*instance
	logger *zap.Logger
}

// NewService discovers every plugin in dir, loads and validates its
// manifest and template, performs one blocking initial fetch per plugin
// (bounded by a short timeout so a slow/unreachable upstream never delays
// server startup for long), and starts each plugin's background refresh
// loop. A dir that does not exist is not an error — plugins are opt-in.
func NewService(dir string, logger *zap.Logger) (*Service, error) {
	paths, err := discover(dir)
	if err != nil {
		return nil, err
	}

	svc := &Service{
		byPage: make(map[string]map[string]*instance),
		logger: logger,
	}

	seenNames := make(map[string]string)   // name -> manifest path
	seenTargets := make(map[string]string) // "page/contentKey" -> manifest path
	for _, path := range paths {
		m, err := loadManifest(path)
		if err != nil {
			return nil, err
		}
		if prev, ok := seenNames[m.Name]; ok {
			return nil, fmt.Errorf("plugin name %q used by both %s and %s", m.Name, prev, path)
		}
		seenNames[m.Name] = path

		target := m.Render.Into.Page + "/" + m.Render.Into.ContentKey
		if prev, ok := seenTargets[target]; ok {
			return nil, fmt.Errorf("render target page=%q contentKey=%q claimed by both %s and %s",
				m.Render.Into.Page, m.Render.Into.ContentKey, prev, path)
		}
		seenTargets[target] = path

		tmpl, err := loadTemplate(m)
		if err != nil {
			return nil, err
		}

		in := newInstance(m, tmpl, logger.With(zap.String("plugin", m.Name)))

		// Bounded blocking first fetch: best-effort so the first page render
		// already has data, but never lets one slow plugin hold up startup.
		startCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		in.refreshOnce(startCtx)
		cancel()

		go in.run()

		svc.instances = append(svc.instances, in)
		if svc.byPage[m.Render.Into.Page] == nil {
			svc.byPage[m.Render.Into.Page] = make(map[string]*instance)
		}
		svc.byPage[m.Render.Into.Page][m.Render.Into.ContentKey] = in
	}

	if len(paths) > 0 {
		logger.Info("plugins loaded", zap.Int("count", len(paths)), zap.String("dir", dir))
	}

	return svc, nil
}

// ContentFor returns the content-key → rendered-HTML overlay for the given
// page ID. Empty map if the page has no plugins targeting it. Safe to call
// concurrently from every request.
func (s *Service) ContentFor(pageID string) map[string]string {
	targets := s.byPage[pageID]
	if len(targets) == 0 {
		return nil
	}
	out := make(map[string]string, len(targets))
	for key, in := range targets {
		if html := in.content(); html != "" {
			out[key] = html
		}
	}
	return out
}

// ImgSrcHosts returns the deduplicated union of every plugin's csp.imgSrc,
// for wiring into the security headers middleware's Content-Security-Policy.
func (s *Service) ImgSrcHosts() []string {
	seen := make(map[string]struct{})
	var hosts []string
	for _, in := range s.instances {
		for _, h := range in.manifest.CSP.ImgSrc {
			if _, ok := seen[h]; !ok {
				seen[h] = struct{}{}
				hosts = append(hosts, h)
			}
		}
	}
	return hosts
}

// Stop halts every plugin's background refresh loop. Called during graceful
// shutdown, mirroring analytics.Service.Close().
func (s *Service) Stop() {
	for _, in := range s.instances {
		in.shutdown()
	}
}
