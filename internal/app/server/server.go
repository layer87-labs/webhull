package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/pkg/analytics"
	"github.com/layer87-labs/webhull/internal/pkg/assets"
	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/consent"
	"github.com/layer87-labs/webhull/internal/pkg/content"
	"github.com/layer87-labs/webhull/internal/pkg/forms"
	"github.com/layer87-labs/webhull/internal/pkg/gate"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
	"github.com/layer87-labs/webhull/internal/pkg/middleware"
	"github.com/layer87-labs/webhull/internal/pkg/navigation"
	"github.com/layer87-labs/webhull/internal/pkg/pages"
	"github.com/layer87-labs/webhull/internal/pkg/security"
	"github.com/layer87-labs/webhull/internal/pkg/seo"
	"github.com/layer87-labs/webhull/internal/pkg/staticassets"
)

// Server is the main application server that wires all modules together.
type Server struct {
	cfg       *config.SiteConfig
	router    *gin.Engine
	httpSrv   *http.Server
	healthSrv *http.Server
	logger    *zap.Logger

	// Domain services
	I18n       *i18n.Service
	Pages      *pages.Service
	Navigation *navigation.Service
	Forms      *forms.Service
	Consent    *consent.Service
	Analytics  *analytics.Service
	SEO        *seo.Service
	Bot        *security.BotDetector
	Assets     *assets.Service
	Gate       *gate.Service // nil when gate is disabled
	ArconGate  *gate.Service // nil when arcon gate is disabled

	// gateLimiter is a dedicated rate limiter for /gate POST submissions.
	gateLimiter *security.RateLimiter

	// arconGateLimiter is a dedicated rate limiter for /arcon/gate POST submissions.
	arconGateLimiter *security.RateLimiter

	// healthStartedAt is used for simple uptime metrics on the health server.
	healthStartedAt time.Time
}

// New creates a new server from configuration.
func New(cfg *config.SiteConfig, logger *zap.Logger) (*Server, error) {
	// Set gin mode
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	s := &Server{
		cfg:    cfg,
		router: router,
		logger: logger,
	}

	// Initialize domain services
	if err := s.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Initialize health server (if enabled)
	if cfg.Health.Enabled != nil && *cfg.Health.Enabled {
		s.healthStartedAt = time.Now().UTC()

		healthPath := defaultPath(cfg.Health.HealthPath, "/health")
		readyPath := defaultPath(cfg.Health.ReadyPath, "/ready")
		metricsPath := defaultPath(cfg.Health.MetricsPath, "/metrics")

		mux := http.NewServeMux()
		mux.HandleFunc(healthPath, s.healthHandler)
		mux.HandleFunc(readyPath, s.readyHandler)
		mux.HandleFunc(metricsPath, s.metricsHandler)

		healthAddr := fmt.Sprintf("%s:%d", cfg.Health.Host, cfg.Health.Port)
		s.healthSrv = &http.Server{
			Addr:    healthAddr,
			Handler: mux,
		}
		if cfg.Health.Timeout > 0 {
			s.healthSrv.ReadTimeout = cfg.Health.Timeout
			s.healthSrv.WriteTimeout = cfg.Health.Timeout
			s.healthSrv.IdleTimeout = cfg.Health.Timeout
		}
	} else {
		logger.Debug("health server disabled")
	}

	//
	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// initServices wires all domain services.
func (s *Server) initServices() error {
	// i18n
	s.I18n = i18n.NewService(s.cfg.I18n.DefaultLanguage, s.cfg.I18n.Languages)

	// Load content files (HTML with YAML frontmatter)
	if s.cfg.ContentDir != "" {
		contentPages, err := content.Load(s.cfg.ContentDir, s.cfg.I18n.Languages, s.logger)
		if err != nil {
			return fmt.Errorf("content loading: %w", err)
		}
		// Merge content pages into any supplementary yaml pages that share the same ID
		// (e.g. a pages.yaml entry that only carries SEO JSON-LD). Content data always
		// wins for i18n/template fields; yaml-only fields (JSONLD) are preserved.
		s.cfg.Pages = mergeContentPages(s.cfg.Pages, contentPages)
	}

	if len(s.cfg.Pages) == 0 {
		return fmt.Errorf("no pages defined — check contentDir setting")
	}

	// Pages
	var err error
	s.Pages, err = pages.NewService(s.cfg.Pages, s.cfg.I18n.Languages)
	if err != nil {
		return fmt.Errorf("pages service: %w", err)
	}

	// Navigation
	s.Navigation = navigation.NewService(s.cfg.Navigation)

	// Consent
	s.Consent = consent.NewService(s.cfg.Consent)

	// SEO
	s.SEO = seo.NewService(s.cfg.Site, s.cfg.SEO, s.I18n)

	// Bot detector
	s.Bot = security.NewBotDetector()

	// Assets (cache-busting hashes): scan embedded built-ins first, then
	// user staticDir so user files always take precedence.
	s.Assets = assets.NewService(s.cfg.Server.StaticDir, "/static", s.logger)
	s.Assets.ScanEmbedded(staticassets.FS, "js", "/static")
	s.Assets.ScanEmbedded(staticassets.FS, "css", "/static")

	// Forms (if contact is enabled)
	if s.cfg.Contact.Enabled {
		mailer := forms.NewSMTPMailer(
			s.cfg.Mail.Host,
			s.cfg.Mail.Port,
			s.cfg.Mail.Username,
			s.cfg.Mail.Password,
			s.cfg.Mail.From,
			s.cfg.Mail.FromName,
			s.cfg.Mail.UseTLS,
		)

		limiter := security.NewRateLimiter(security.RateLimitConfig{
			Limit:       s.cfg.Contact.RateLimit.Requests,
			Window:      s.cfg.Contact.RateLimit.Window,
			CleanupTick: 5 * time.Minute,
		}, s.logger)

		s.Forms = forms.NewService(s.cfg.Contact, s.cfg.Mail, mailer, limiter, s.logger)
	}

	// Analytics
	s.initAnalytics()

	// Access Gate (optional)
	if s.cfg.Gate.Enabled {
		isProduction := s.cfg.Server.Environment == "production"
		s.Gate = gate.NewService(s.cfg.Gate, s.logger, isProduction)
		s.gateLimiter = security.NewRateLimiter(security.RateLimitConfig{
			Limit:       3,
			Window:      15 * time.Minute,
			CleanupTick: 5 * time.Minute,
		}, s.logger)
	}

	// Arcon Gate (optional) — protects only /arcon/*
	if s.cfg.ArconGate.Enabled {
		isProduction := s.cfg.Server.Environment == "production"
		arconGateCfg := config.GateConfig{
			Enabled:      true,
			CookieName:   s.cfg.ArconGate.CookieName,
			CookieMaxAge: s.cfg.ArconGate.CookieMaxAge,
			CookieSecret: s.cfg.ArconGate.CookieSecret,
			Codes:        s.cfg.ArconGate.Codes,
		}
		s.ArconGate = gate.NewService(arconGateCfg, s.logger, isProduction)
		s.arconGateLimiter = security.NewRateLimiter(security.RateLimitConfig{
			Limit:       3,
			Window:      15 * time.Minute,
			CleanupTick: 5 * time.Minute,
		}, s.logger)
	}

	return nil
}

// mergeContentPages merges content-loaded pages into the base slice (from pages.yaml).
// When a content page shares an ID with a base entry, the content data (i18n, template,
// per-file SEO) is overlaid onto the base entry so that yaml-only fields (e.g. JSONLD)
// are preserved. Content pages with no matching base entry are appended as-is.
func mergeContentPages(base []config.PageConfig, content []config.PageConfig) []config.PageConfig {
	// Build an index of base pages by ID.
	idx := make(map[string]int, len(base))
	for i, p := range base {
		idx[p.ID] = i
	}

	var extra []config.PageConfig
	for _, cp := range content {
		i, ok := idx[cp.ID]
		if !ok {
			extra = append(extra, cp)
			continue
		}

		// Overlay i18n data from the content page onto the base entry.
		if base[i].I18n == nil {
			base[i].I18n = make(map[string]config.PageI18nConfig)
		}
		for lang, i18nCfg := range cp.I18n {
			base[i].I18n[lang] = i18nCfg
		}

		// Template comes from content if not already set in yaml.
		if base[i].Template == "" {
			base[i].Template = cp.Template
		}

		// Merge per-file SEO fields (content wins; JSONLD from yaml is preserved).
		if cp.SEO.Priority > 0 {
			base[i].SEO.Priority = cp.SEO.Priority
		}
		if cp.SEO.ChangeFreq != "" {
			base[i].SEO.ChangeFreq = cp.SEO.ChangeFreq
		}
		if cp.SEO.OGImage != "" {
			base[i].SEO.OGImage = cp.SEO.OGImage
		}
		if cp.SEO.OGType != "" {
			base[i].SEO.OGType = cp.SEO.OGType
		}
		if cp.SEO.NoIndex {
			base[i].SEO.NoIndex = true
		}
	}

	return append(base, extra...)
}

// initAnalytics sets up analytics providers.
func (s *Server) initAnalytics() {
	var providers []analytics.Provider

	if s.cfg.Analytics.Plausible != nil && s.cfg.Analytics.Plausible.Enabled {
		providers = append(providers, analytics.NewPlausibleProvider(
			s.cfg.Analytics.Plausible.BaseURL,
			s.cfg.Analytics.Plausible.ScriptPath,
			s.cfg.Analytics.Plausible.Domain,
			s.logger,
		))
	}

	if s.cfg.Analytics.Collector != nil && s.cfg.Analytics.Collector.Enabled {
		providers = append(providers, analytics.NewCollectorProvider(
			s.cfg.Analytics.Collector.Endpoint,
			s.logger,
		))
	}

	s.Analytics = analytics.NewService(s.logger, providers...)
}

// setupMiddleware configures the middleware stack.
func (s *Server) setupMiddleware() {
	isProduction := s.cfg.Server.Environment == "production"

	// Build security headers config with CSP awareness.
	secCfg := security.SecurityHeadersConfig{
		IsProduction: isProduction,
	}
	if s.cfg.Analytics.Plausible != nil && s.cfg.Analytics.Plausible.Enabled {
		secCfg.PlausibleBaseURL = s.cfg.Analytics.Plausible.BaseURL
	}
	if s.cfg.ArconGate.ContentDir != "" {
		secCfg.ArconPathPrefix = "/arcon"
	}

	s.router.Use(gin.Recovery())
	s.router.Use(middleware.GzipMiddleware())
	s.router.Use(middleware.LoggingMiddleware(s.logger, 1*time.Second))
	s.router.Use(middleware.CacheMiddleware(s.cfg.Server.CacheMaxAge, s.cfg.Server.StaticCacheMaxAge))
	s.router.Use(security.SecurityHeadersMiddleware(secCfg))
	s.router.Use(s.I18n.Middleware())
	s.router.Use(s.Consent.Middleware())
}

// setupRoutes registers all routes.
func (s *Server) setupRoutes() {
	// --- Always-public routes ---

	// SEO
	s.router.GET("/sitemap.xml", s.SEO.ServeSitemap(s.Pages.All()))
	s.router.GET("/robots.txt", s.SEO.ServeRobotsTxt())

	// Static files: serve webhull's embedded built-ins, then overlay the
	// user's staticDir on top so project files always take precedence.
	embeddedStatic := http.FS(staticassets.WrapFS())
	if s.cfg.Server.StaticDir != "" {
		s.router.StaticFS("/static", staticassets.OverlayFS(s.cfg.Server.StaticDir, embeddedStatic))
	} else {
		s.router.StaticFS("/static", embeddedStatic)
	}

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Gate login page (public — otherwise the gate page itself would be blocked)
	if s.Gate != nil {
		s.router.GET("/gate", s.handleGatePage())
		s.router.POST("/gate", s.handleGateSubmit())
	}

	// Arcon content routes.
	// When gate is enabled: gate login + protected content.
	// When gate is disabled but contentDir is set: public content serving.
	if s.ArconGate != nil {
		arconGroup := s.router.Group("/arcon")
		arconGroup.POST("/gate", s.handleArconGateSubmit())
		arconGroup.GET("/*filepath", s.handleArconGateOrContent())
		arconGroup.HEAD("/*filepath", s.handleArconGateOrContent())
	} else if s.cfg.ArconGate.ContentDir != "" {
		// No gate — serve arcon content publicly as static files.
		fs := http.FileServer(http.Dir(s.cfg.ArconGate.ContentDir))
		serveArcon := func(c *gin.Context) {
			c.Request.URL.Path = c.Param("filepath")
			fs.ServeHTTP(c.Writer, c.Request)
		}
		arconGroup := s.router.Group("/arcon")
		arconGroup.GET("/*filepath", serveArcon)
		arconGroup.HEAD("/*filepath", serveArcon)
	}

	// --- Protected routes (wrapped in GateMiddleware when gate is active) ---
	protected := s.protectedRouter()

	// Root route: single-page mode renders directly; multi-page mode redirects to start slug.
	if s.Pages.HasRootPages() {
		protected.GET("/", s.handleRootPage())
		protected.HEAD("/", s.handleRootPage())
	} else {
		protected.GET("/", s.I18n.RootRedirect(s.Pages.StartSlugs()))
		protected.HEAD("/", s.I18n.RootRedirect(s.Pages.StartSlugs()))
	}

	// Page routes — register all slugs from all languages
	for _, slug := range s.Pages.Slugs() {
		slug := slug // capture
		if slug == "" {
			continue // root already registered above
		}
		protected.GET("/"+slug, s.handlePage(slug))
	}

	// Contact form endpoint
	protected.POST("/api/contact", s.handleContact())

	// Analytics event endpoint
	protected.POST("/api/events", s.handleAnalyticsEvent())

	// Plausible proxy routes (first-party analytics, avoids ad-blockers)
	if s.cfg.Analytics.Plausible != nil && s.cfg.Analytics.Plausible.Enabled {
		protected.GET("/js/script.js", s.handlePlausibleScript())
		protected.POST("/api/event", s.handlePlausibleEvent())
	}

	// Consent update endpoint
	protected.POST("/api/consent", s.handleConsentUpdate())

	// 404 handler
	s.router.NoRoute(s.handleNotFound())
}

// protectedRouter returns a Gin IRouter that enforces gate authentication when
// the gate is enabled. When the gate is disabled it returns the base router
// unchanged so there is zero overhead.
func (s *Server) protectedRouter() gin.IRouter {
	if s.Gate == nil {
		return s.router
	}
	group := s.router.Group("/")
	group.Use(gate.Middleware(s.Gate, "/gate"))
	return group
}

// Start runs the HTTP server with graceful shutdown.
func (s *Server) Start() error {
	addr := s.cfg.Server.Host + ":" + s.cfg.Server.Port

	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: s.cfg.Server.WriteTimeout,
		IdleTimeout:  s.cfg.Server.IdleTimeout,
	}

	// Start health server (if enabled)
	if s.healthSrv != nil {
		go func() {
			s.logger.Info("health server starting", zap.String("address", s.healthSrv.Addr))
			if err := s.healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.logger.Error("health server failed", zap.Error(err))
			}
		}()
	}

	// Start main HTTP server in goroutine
	go func() {
		s.logger.Info("server starting",
			zap.String("address", addr),
			zap.String("environment", s.cfg.Server.Environment))

		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Server.ShutdownTimeout)
	defer cancel()

	// Cleanup
	s.Analytics.Close()

	// Shutdown health server
	if s.healthSrv != nil {
		if err := s.healthSrv.Shutdown(ctx); err != nil {
			s.logger.Error("health server shutdown failed", zap.Error(err))
		}
	}

	// Shutdown main HTTP server
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		s.logger.Error("forced shutdown", zap.Error(err))
		return err
	}

	s.logger.Info("server stopped")
	return nil
}

// Router returns the Gin engine for testing.
func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	serviceName := strings.TrimSpace(s.cfg.Health.ServiceName)
	if serviceName == "" {
		serviceName = "web-core"
	}

	serviceVersion := strings.TrimSpace(s.cfg.Health.ServiceVersion)
	if serviceVersion == "" {
		serviceVersion = "unknown"
	}

	uptime := time.Since(s.healthStartedAt).Seconds()
	if s.healthStartedAt.IsZero() {
		uptime = 0
	}

	// Minimal Prometheus-style metrics output without external dependencies.
	metrics := fmt.Sprintf(
		"# HELP webcore_info Static build and service metadata.\n"+
			"# TYPE webcore_info gauge\n"+
			"webcore_info{service=\"%s\",version=\"%s\"} 1\n"+
			"# HELP webcore_uptime_seconds Process uptime in seconds.\n"+
			"# TYPE webcore_uptime_seconds gauge\n"+
			"webcore_uptime_seconds %.0f\n",
		serviceName,
		serviceVersion,
		uptime,
	)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(metrics))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func defaultPath(path, fallback string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return fallback
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}
