package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/app/templates"
	"github.com/layer87-labs/webhull/internal/pkg/analytics"
	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/consent"
	"github.com/layer87-labs/webhull/internal/pkg/forms"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
	"github.com/layer87-labs/webhull/internal/pkg/pages"
	"github.com/layer87-labs/webhull/internal/pkg/seo"
)

// handleRootPage serves the single root page in single-page mode.
// It renders the "home" page (with empty slug) in the request's detected language.
func (s *Server) handleRootPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Detect language from i18n middleware context.
		lang := s.I18n.Default()
		if lc, exists := c.Get(i18n.ContextKey); exists {
			if langCtx, ok := lc.(*i18n.LanguageContext); ok {
				lang = langCtx.Current
			}
		}

		page := s.Pages.RootPage(lang)
		if page == nil {
			// Fallback to default language root page.
			page = s.Pages.RootPage(s.I18n.Default())
		}
		if page == nil {
			s.handleNotFound()(c)
			return
		}

		// Update i18n context with this page's language and alternates.
		if lc, exists := c.Get(i18n.ContextKey); exists {
			if langCtx, ok := lc.(*i18n.LanguageContext); ok {
				langCtx.Current = page.Language
				langCtx.Alternates = page.Alternates
			}
		}

		// Persist language choice as cookie.
		c.SetCookie(i18n.CookieName, page.Language.String(), 30*24*3600, "/", "", true, false)

		consentState := consent.StateFromContext(c)
		data := s.buildPageData(page, consentState)

		html, err := renderTemplate(c.Request.Context(), page.Template, data)
		if err != nil {
			s.logger.Error("single-page render failed",
				zap.String("template", page.Template),
				zap.Error(err))
			c.String(http.StatusInternalServerError, "render error")
			return
		}

		hash := sha256.Sum256(html)
		etag := fmt.Sprintf(`"%x"`, hash[:8])
		c.Header("ETag", etag)
		if match := c.GetHeader("If-None-Match"); match == etag {
			c.Status(http.StatusNotModified)
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", html)

		s.Analytics.TrackServerSide(
			consentState,
			s.cfg.Site.BaseURL+"/",
			c.ClientIP(),
			c.Request.UserAgent(),
			c.GetHeader("Accept-Language"),
		)
	}
}

// handlePage serves a pre-rendered or dynamically rendered page.
func (s *Server) handlePage(slug string) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := s.Pages.Resolve(slug)
		if page == nil {
			s.handleNotFound()(c)
			return
		}

		// Update the i18n context with alternates for this page
		lc, _ := c.Get(i18n.ContextKey)
		if langCtx, ok := lc.(*i18n.LanguageContext); ok {
			langCtx.Current = page.Language
			langCtx.Alternates = page.Alternates
		}

		// Update language cookie to match this page's language
		c.SetCookie(i18n.CookieName, page.Language.String(), 30*24*3600, "/", "", true, false)

		// Build template data
		consentState := consent.StateFromContext(c)
		data := s.buildPageData(page, consentState)

		// Render the page template
		html, err := renderTemplate(c.Request.Context(), page.Template, data)
		if err != nil {
			s.logger.Error("template render failed",
				zap.String("slug", slug),
				zap.String("template", page.Template),
				zap.Error(err))
			c.String(http.StatusInternalServerError, "render error")
			return
		}

		// ETag: hash the rendered HTML so browsers can validate freshness.
		hash := sha256.Sum256(html)
		etag := fmt.Sprintf(`"%x"`, hash[:8])
		c.Header("ETag", etag)

		// If the client already has this version, return 304.
		if match := c.GetHeader("If-None-Match"); match == etag {
			c.Status(http.StatusNotModified)
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", html)

		// Server-side pageview tracking (when client-side JS is not active)
		s.Analytics.TrackServerSide(
			consentState,
			s.cfg.Site.BaseURL+"/"+slug,
			c.ClientIP(),
			c.Request.UserAgent(),
			c.GetHeader("Accept-Language"),
		)
	}
}

// buildPageData assembles the complete view model for a page.
func (s *Server) buildPageData(page *pages.Page, consentState *consent.State) *templates.PageData {
	meta := s.SEO.BuildMetaTags(page)
	header := s.Navigation.ResolveHeader(page.Language, page.Slug)
	footer := s.Navigation.ResolveFooter(page.Language, page.Slug)
	copyright := s.SEO.Copyright()
	langLinks := s.SEO.LanguageSwitchLinks(page)

	// Build consent banner data
	var consentBanner *templates.ConsentBannerData
	if s.Consent.IsEnabled() {
		consentBanner = &templates.ConsentBannerData{
			Enabled:    true,
			Texts:      s.Consent.Texts(page.Language),
			Categories: s.Consent.Categories(page.Language),
		}
	}

	// Build analytics data
	analyticsData := templates.AnalyticsData{}
	if s.cfg.Analytics.Plausible != nil && s.cfg.Analytics.Plausible.Enabled {
		analyticsData.PlausibleEnabled = true
		analyticsData.PlausibleDomain = s.cfg.Analytics.Plausible.Domain
		analyticsData.PlausibleScript = s.cfg.Analytics.Plausible.BaseURL + s.cfg.Analytics.Plausible.ScriptPath
	}
	if s.cfg.Analytics.Collector != nil && s.cfg.Analytics.Collector.Enabled {
		analyticsData.CollectorEnabled = true
	}

	return &templates.PageData{
		Page:          page,
		Meta:          meta,
		Header:        header,
		Footer:        footer,
		Copyright:     copyright.String(),
		LangLinks:     langLinks,
		Consent:       consentState,
		ConsentConfig: consentBanner,
		Site: templates.SiteData{
			Name:          s.cfg.Site.Name,
			BaseURL:       s.cfg.Site.BaseURL,
			LogoPath:      s.cfg.Site.LogoPath,
			FaviconPath:   s.cfg.Site.FaviconPath,
			ShowLangFlags: s.cfg.Site.ShowLangFlags,
		},
		UI:             s.resolveUI(page.Language),
		Analytics:      analyticsData,
		IsBot:          s.Bot.IsBot(""), // will be set per-request below
		ContactEnabled: s.cfg.Contact.Enabled,
		Assets:         s.Assets,
		InstagramFeed:  s.buildInstagramFeed(),
	}
}

// buildInstagramFeed builds Instagram feed data from the instagram service.
// Returns nil when the service is disabled or has no posts.
func (s *Server) buildInstagramFeed() *templates.InstagramFeedData {
	if s.Instagram == nil || !s.Instagram.HasPosts() {
		return nil
	}

	posts := s.Instagram.GetPosts(context.Background())
	if len(posts) == 0 {
		return nil
	}

	username := ""
	if len(posts) > 0 {
		username = posts[0].Username
	}

	tmplPosts := make([]templates.InstagramPost, 0, len(posts))
	for _, p := range posts {
		tmplPosts = append(tmplPosts, templates.InstagramPost{
			ID:            p.ID,
			Caption:       p.Caption,
			MediaType:     p.MediaType,
			MediaURL:      p.MediaURL,
			Permalink:     p.Permalink,
			Timestamp:     p.Timestamp,
			Username:      p.Username,
			LikeCount:     p.LikeCount,
			CommentsCount: p.CommentsCount,
		})
	}

	return &templates.InstagramFeedData{
		Posts:          tmplPosts,
		Username:       username,
		ProfileURL:     fmt.Sprintf("https://instagram.com/%s", username),
		ShowEngagement: true,
	}
}

// handleContact processes contact form submissions.
func (s *Server) handleContact() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req forms.ContactRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, forms.ContactResponse{
				Success: false,
				Message: "invalid request",
			})
			return
		}

		// Resolve language from context
		lang := s.I18n.Default()
		if lc, exists := c.Get(i18n.ContextKey); exists {
			if langCtx, ok := lc.(*i18n.LanguageContext); ok {
				lang = langCtx.Current
			}
		}

		// Resolve field definitions: site config → fallback to built-in defaults
		fieldDefs := templates.DefaultFields(lang.String())
		if uiCfg, ok := s.cfg.UI[lang.String()]; ok && len(uiCfg.ContactForm.Fields) > 0 {
			fieldDefs = uiCfg.ContactForm.Fields
		}

		resp, err := s.Forms.Submit(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent(), lang, fieldDefs)
		if err != nil {
			s.logger.Error("contact form error", zap.Error(err))
			c.JSON(http.StatusTooManyRequests, resp)
			return
		}

		if resp.Success {
			c.JSON(http.StatusOK, resp)
		} else {
			c.JSON(http.StatusBadRequest, resp)
		}
	}
}

// handleAnalyticsEvent receives tracking events from the JS snippet.
func (s *Server) handleAnalyticsEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		var event analytics.Event
		if err := c.ShouldBindJSON(&event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event"})
			return
		}

		consentState := consent.StateFromContext(c)

		s.Analytics.Track(
			c.Request.Context(),
			consentState,
			event,
			c.ClientIP(),
			c.Request.UserAgent(),
			c.GetHeader("Accept-Language"),
		)

		c.JSON(http.StatusAccepted, gin.H{"status": "ok"})
	}
}

// handleConsentUpdate allows the client to update consent preferences.
func (s *Server) handleConsentUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {
		var state consent.State
		if err := c.ShouldBindJSON(&state); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid consent state"})
			return
		}

		state.Decided = true

		// Serialize and set cookie
		c.SetCookie(consent.CookieName, mustJSON(state), 365*24*3600, "/", "", true, false)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// handlePlausibleScript proxies the Plausible tracking script through our domain.
// This avoids ad-blockers and keeps all traffic first-party.
func (s *Server) handlePlausibleScript() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := s.plausibleProvider()
		if provider == nil {
			c.Status(http.StatusNotFound)
			return
		}

		data, err := provider.ProxyScript(c.Request.Context())
		if err != nil {
			s.logger.Warn("plausible script proxy failed", zap.Error(err))
			c.Status(http.StatusBadGateway)
			return
		}

		c.Header("Content-Type", "application/javascript; charset=utf-8")
		c.Header("Cache-Control", "public, max-age=1800")
		c.Data(http.StatusOK, "application/javascript", data)
	}
}

// handlePlausibleEvent proxies analytics events to Plausible, preserving
// the original client IP and user agent for accurate analytics.
func (s *Server) handlePlausibleEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := s.plausibleProvider()
		if provider == nil {
			c.Status(http.StatusNotFound)
			return
		}

		// Read raw body — Plausible script sends Content-Type: text/plain
		// with its own JSON structure (n, u, d, etc.), not our Event struct.
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
			return
		}

		// Check analytics consent
		consentState := consent.StateFromContext(c)
		if consentState != nil && !consentState.IsAllowed("analytics") {
			c.JSON(http.StatusOK, gin.H{"status": "skipped"})
			return
		}

		status, err := provider.ProxyEvent(
			c.Request.Context(),
			body,
			c.ClientIP(),
			c.Request.UserAgent(),
			c.GetHeader("Accept-Language"),
		)
		if err != nil {
			s.logger.Warn("plausible event proxy failed", zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "proxy failed"})
			return
		}

		c.Status(status)
	}
}

// plausibleProvider returns the Plausible provider if configured, nil otherwise.
func (s *Server) plausibleProvider() *analytics.PlausibleProvider {
	if s.cfg.Analytics.Plausible == nil || !s.cfg.Analytics.Plausible.Enabled {
		return nil
	}
	for _, p := range s.Analytics.Providers() {
		if pp, ok := p.(*analytics.PlausibleProvider); ok {
			return pp
		}
	}
	return nil
}

// resolveUI returns the UI strings for the given language.
func (s *Server) resolveUI(lang i18n.Language) config.UIStringsConfig {
	if ui, ok := s.cfg.UI[lang.String()]; ok {
		return ui
	}
	// Fallback to default language
	if ui, ok := s.cfg.UI[s.cfg.I18n.DefaultLanguage]; ok {
		return ui
	}
	return config.UIStringsConfig{}
}

func mustJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// handleNotFound renders a styled 404 page with the full site layout.
func (s *Server) handleNotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Detect language from context (set by i18n middleware)
		lang := s.I18n.Default()
		if lc, exists := c.Get(i18n.ContextKey); exists {
			if langCtx, ok := lc.(*i18n.LanguageContext); ok {
				lang = langCtx.Current
			}
		}

		ui := s.resolveUI(lang)

		// Defaults if not configured
		title := ui.NotFoundTitle
		if title == "" {
			if lang == i18n.LangEN {
				title = "Page not found"
			} else {
				title = "Seite nicht gefunden"
			}
		}
		subtitle := ui.NotFoundSubtitle
		if subtitle == "" {
			if lang == i18n.LangEN {
				subtitle = "The page you are looking for does not exist or has been moved."
			} else {
				subtitle = "Die gesuchte Seite existiert nicht oder wurde verschoben."
			}
		}
		button := ui.NotFoundButton
		if button == "" {
			if lang == i18n.LangEN {
				button = "Back to Home"
			} else {
				button = "Zur Startseite"
			}
		}

		// Build a minimal page for the 404 template
		notFoundPage := &pages.Page{
			ID:       "notfound",
			Template: "notfound",
			Title:    "404",
			Language: lang,
			Content: map[string]string{
				"notFoundTitle":    title,
				"notFoundSubtitle": subtitle,
				"notFoundButton":   button,
			},
		}

		consentState := consent.StateFromContext(c)
		header := s.Navigation.ResolveHeader(lang, "")
		footer := s.Navigation.ResolveFooter(lang, "")
		copyright := s.SEO.Copyright()

		data := &templates.PageData{
			Page:      notFoundPage,
			Meta:      seo.MetaTags{Title: "404 – " + s.cfg.Site.Name},
			Header:    header,
			Footer:    footer,
			Copyright: copyright.String(),
			Consent:   consentState,
			ConsentConfig: func() *templates.ConsentBannerData {
				if s.Consent.IsEnabled() {
					return &templates.ConsentBannerData{
						Enabled:    true,
						Texts:      s.Consent.Texts(lang),
						Categories: s.Consent.Categories(lang),
					}
				}
				return nil
			}(),
			Site: templates.SiteData{
				Name:          s.cfg.Site.Name,
				BaseURL:       s.cfg.Site.BaseURL,
				LogoPath:      s.cfg.Site.LogoPath,
				FaviconPath:   s.cfg.Site.FaviconPath,
				ShowLangFlags: s.cfg.Site.ShowLangFlags,
			},
			UI:     ui,
			Assets: s.Assets,
		}

		html, err := renderTemplate(c.Request.Context(), "notfound", data)
		if err != nil {
			s.logger.Error("404 template render failed", zap.Error(err))
			c.String(http.StatusNotFound, "404 – page not found")
			return
		}

		c.Data(http.StatusNotFound, "text/html; charset=utf-8", html)
	}
}

// handleGatePage renders the access gate login page.
//
// GET /gate              → renders gate page (no error)
// GET /gate?redirect=... → renders gate page, threads redirect through hidden form field
func (s *Server) handleGatePage() gin.HandlerFunc {
	return func(c *gin.Context) {
		redirect := sanitizeRedirect(c.Query("redirect"))

		data := &templates.GatePageData{
			SiteName:   s.cfg.Site.Name,
			LogoPath:   s.cfg.Site.LogoPath,
			Redirect:   redirect,
			FormAction: "/gate",
		}

		s.renderGatePage(c, data, http.StatusOK)
	}
}

// handleGateSubmit processes the access code form.
//
// POST /gate  (application/x-www-form-urlencoded)
//
//	Fields: code, redirect (optional)
//
// On success: sets signed session cookie, redirects to redirect or "/".
// On failure: re-renders gate page with error message.
// Rate limited: 3 attempts per IP per 15 minutes.
func (s *Server) handleGateSubmit() gin.HandlerFunc {
	msgTooMany := s.cfg.Gate.MsgTooManyAttempts
	if msgTooMany == "" {
		msgTooMany = "Too many attempts. Please try again later."
	}
	msgInvalid := s.cfg.Gate.MsgInvalidCode
	if msgInvalid == "" {
		msgInvalid = "Invalid access code. Please try again."
	}

	return func(c *gin.Context) {
		// Rate limiting — checked before anything else.
		ip := c.ClientIP()
		if !s.gateLimiter.IsAllowed(ip) {
			data := &templates.GatePageData{
				SiteName:   s.cfg.Site.Name,
				LogoPath:   s.cfg.Site.LogoPath,
				Error:      true,
				ErrorMsg:   msgTooMany,
				Redirect:   sanitizeRedirect(c.PostForm("redirect")),
				FormAction: "/gate",
			}
			s.renderGatePage(c, data, http.StatusTooManyRequests)
			return
		}

		code := strings.TrimSpace(c.PostForm("code"))
		redirect := sanitizeRedirect(c.PostForm("redirect"))

		_, ok := s.Gate.ValidateCode(code)
		if !ok {
			s.logger.Debug("gate: invalid code attempt", zap.String("ip", ip))
			data := &templates.GatePageData{
				SiteName:   s.cfg.Site.Name,
				LogoPath:   s.cfg.Site.LogoPath,
				Error:      true,
				ErrorMsg:   msgInvalid,
				Redirect:   redirect,
				FormAction: "/gate",
			}
			s.renderGatePage(c, data, http.StatusUnauthorized)
			return
		}

		// Code accepted — issue session cookie.
		s.Gate.CreateSessionCookie(c)

		// Redirect to the originally requested URL or fall back to root.
		destination := "/"
		if redirect != "" {
			destination = redirect
		}
		c.Redirect(http.StatusFound, destination)
	}
}

// renderGatePage renders the gate template directly without the page layout.
func (s *Server) renderGatePage(c *gin.Context, data *templates.GatePageData, status int) {
	component := templates.GatePage(data)
	var buf bytes.Buffer
	if err := component.Render(c.Request.Context(), &buf); err != nil {
		s.logger.Error("gate template render failed", zap.Error(err))
		c.String(http.StatusInternalServerError, "render error")
		return
	}
	c.Data(status, "text/html; charset=utf-8", buf.Bytes())
}

// handleArconGateOrContent handles all GET /arcon/* requests.
//
// Gin does not allow mixing a specific child route (/arcon/gate) with a wildcard
// route (/arcon/*filepath) in the same radix tree, so both cases are handled here:
//
//   - /arcon/gate              → renders the gate login page (always public)
//   - /arcon/gate?redirect=... → same, threads redirect through the login form
//   - /arcon/<anything>        → checks arcon session cookie; redirects to gate on miss;
//     serves the static file from ArconGate.ContentDir on hit
func (s *Server) handleArconGateOrContent() gin.HandlerFunc {
	fs := http.FileServer(http.Dir(s.cfg.ArconGate.ContentDir))

	return func(c *gin.Context) {
		fp := c.Param("filepath") // e.g. "/gate", "/index.html", "/"

		// Gate login page — always public, no cookie required.
		if fp == "/gate" || fp == "/gate/" {
			redirect := sanitizeArconRedirect(c.Query("redirect"))
			data := &templates.GatePageData{
				SiteName:   s.cfg.ArconGate.Title,
				LogoPath:   s.cfg.Site.LogoPath,
				Redirect:   redirect,
				FormAction: "/arcon/gate",
			}
			s.renderGatePage(c, data, http.StatusOK)
			return
		}

		// All other paths require a valid arcon session cookie.
		if !s.ArconGate.IsAuthenticated(c) {
			orig := c.Request.URL.RequestURI()
			target := "/arcon/gate"
			if orig != "" {
				target = "/arcon/gate?redirect=" + url.QueryEscape(orig)
			}
			c.Redirect(http.StatusFound, target)
			c.Abort()
			return
		}

		// Authenticated — serve the static file.
		// Strip /arcon prefix so the file server maps to ContentDir correctly.
		c.Request.URL.Path = fp
		fs.ServeHTTP(c.Writer, c.Request)
	}
}

// handleArconGateSubmit processes the ARCON access code form.
//
// POST /arcon/gate  (application/x-www-form-urlencoded)
//
// On success: sets signed arcon session cookie, redirects to redirect or "/arcon/".
// On failure: re-renders gate page with error message.
// Rate limited: 3 attempts per IP per 15 minutes.
func (s *Server) handleArconGateSubmit() gin.HandlerFunc {
	msgTooMany := s.cfg.Gate.MsgTooManyAttempts
	if msgTooMany == "" {
		msgTooMany = "Too many attempts. Please try again later."
	}
	msgInvalid := s.cfg.Gate.MsgInvalidCode
	if msgInvalid == "" {
		msgInvalid = "Invalid access code. Please try again."
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !s.arconGateLimiter.IsAllowed(ip) {
			data := &templates.GatePageData{
				SiteName:   s.cfg.ArconGate.Title,
				LogoPath:   s.cfg.Site.LogoPath,
				Error:      true,
				ErrorMsg:   msgTooMany,
				Redirect:   sanitizeRedirect(c.PostForm("redirect")),
				FormAction: "/arcon/gate",
			}
			s.renderGatePage(c, data, http.StatusTooManyRequests)
			return
		}

		code := strings.TrimSpace(c.PostForm("code"))
		redirect := sanitizeArconRedirect(c.PostForm("redirect"))

		_, ok := s.ArconGate.ValidateCode(code)
		if !ok {
			s.logger.Debug("arcon gate: invalid code attempt", zap.String("ip", ip))
			data := &templates.GatePageData{
				SiteName:   s.cfg.ArconGate.Title,
				LogoPath:   s.cfg.Site.LogoPath,
				Error:      true,
				ErrorMsg:   msgInvalid,
				Redirect:   redirect,
				FormAction: "/arcon/gate",
			}
			s.renderGatePage(c, data, http.StatusUnauthorized)
			return
		}

		// Code accepted — issue arcon session cookie.
		s.ArconGate.CreateSessionCookie(c)

		destination := "/arcon/"
		if redirect != "" {
			destination = redirect
		}
		c.Redirect(http.StatusFound, destination)
	}
}

// sanitizeRedirect validates a redirect target to prevent open-redirect attacks.
// Only path-only URLs (starting with /) are accepted; anything else is discarded.
func sanitizeRedirect(raw string) string {
	if raw == "" {
		return ""
	}
	// Parse to detect if this is an absolute URL (http://evil.com) and reject it.
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return ""
	}
	// Reject anything with a host (absolute URL).
	if u.Host != "" || u.Scheme != "" {
		return ""
	}
	// Must start with / to be a valid path.
	if !strings.HasPrefix(u.Path, "/") {
		return ""
	}
	// Disallow gate pages to avoid redirect loops.
	if strings.HasPrefix(u.Path, "/gate") || strings.HasPrefix(u.Path, "/arcon/gate") {
		return ""
	}
	return raw
}

// sanitizeArconRedirect validates an arcon-specific redirect target.
// Only paths under /arcon/ are accepted to prevent redirecting outside the experience.
func sanitizeArconRedirect(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return ""
	}
	if u.Host != "" || u.Scheme != "" {
		return ""
	}
	if !strings.HasPrefix(u.Path, "/arcon/") && u.Path != "/arcon" {
		return ""
	}
	if strings.HasPrefix(u.Path, "/arcon/gate") {
		return ""
	}
	return raw
}
