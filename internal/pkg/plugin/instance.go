package plugin

import (
	"context"
	"html/template"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// instance is one running plugin: its manifest, HTTP client, compiled
// render template, and the last-good rendered fragment. Refreshed on a
// ticker in the background — request handlers only ever read cached.
type instance struct {
	manifest *Manifest
	client   *http.Client
	tmpl     *template.Template
	logger   *zap.Logger

	mu       sync.RWMutex
	html     string    // last successfully rendered fragment
	lastGood time.Time // when html was last refreshed successfully
	haveGood bool

	stop chan struct{}
	done chan struct{}
}

func newInstance(m *Manifest, tmpl *template.Template, logger *zap.Logger) *instance {
	return &instance{
		manifest: m,
		client:   &http.Client{Timeout: m.Source.Timeout},
		tmpl:     tmpl,
		logger:   logger,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// refreshOnce fetches, selects and renders once, updating the cache on
// success. On failure it logs and leaves the existing cache untouched —
// callers decide separately whether a stale result is still servable via
// content(), based on StaleWhileError.
func (in *instance) refreshOnce(ctx context.Context) {
	fetchCtx, cancel := context.WithTimeout(ctx, in.manifest.Source.Timeout)
	defer cancel()

	parsed, err := fetch(fetchCtx, in.client, in.manifest.Source)
	if err != nil {
		in.logger.Warn("plugin fetch failed",
			zap.Error(err))
		return
	}

	items, err := selectItems(parsed, in.manifest.Select)
	if err != nil {
		in.logger.Warn("plugin select failed",
			zap.Error(err))
		return
	}

	html, err := renderHTML(in.tmpl, items)
	if err != nil {
		in.logger.Warn("plugin render failed",
			zap.Error(err))
		return
	}

	in.mu.Lock()
	in.html = html
	in.lastGood = time.Now()
	in.haveGood = true
	in.mu.Unlock()

	in.logger.Info("plugin refreshed",
		zap.Int("items", len(items)))
}

// run starts the background refresh loop. Blocks until stop() is called.
func (in *instance) run() {
	defer close(in.done)

	ticker := time.NewTicker(in.manifest.Source.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-in.stop:
			return
		case <-ticker.C:
			in.refreshOnce(context.Background())
		}
	}
}

func (in *instance) shutdown() {
	close(in.stop)
	<-in.done
}

// content returns the cached fragment, or "" if no fetch has ever
// succeeded, or the last good fetch is older than StaleWhileError.
func (in *instance) content() string {
	in.mu.RLock()
	defer in.mu.RUnlock()

	if !in.haveGood {
		return ""
	}
	if time.Since(in.lastGood) > in.manifest.Source.StaleWhileError {
		return ""
	}
	return in.html
}
