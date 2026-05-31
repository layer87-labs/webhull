package consent

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
)

const (
	// CookieName is the consent cookie key.
	CookieName = "consent"

	// ContextKey is the gin context key for consent state.
	ContextKey = "consent"

	// bypassModeAccept accepts all consent categories (banner + consent.js suppressed).
	bypassModeAccept = "accept"

	// bypassModeReject accepts only required consent categories (banner + consent.js suppressed).
	bypassModeReject = "reject"
)

// Service manages cookie consent state.
type Service struct {
	categories []Category
	i18nTexts  map[i18n.Language]config.ConsentI18nConfig
	enabled    bool
}

// NewService creates a consent service from configuration.
func NewService(cfg config.ConsentConfig) *Service {
	categories := make([]Category, 0)
	for key, cat := range cfg.Categories {
		categories = append(categories, Category{
			Key:      key,
			Required: cat.Required,
			Default:  cat.Default,
		})
	}

	i18nTexts := make(map[i18n.Language]config.ConsentI18nConfig)
	for lang, texts := range cfg.I18n {
		i18nTexts[i18n.Language(lang)] = texts
	}

	return &Service{
		categories: categories,
		i18nTexts:  i18nTexts,
		enabled:    cfg.Enabled,
	}
}

// Middleware reads the consent cookie and stores the state in context.
// It also handles the server-side consent bypass for automated tools
// (Lighthouse, Unlighthouse, Playwright, etc.) by inspecting standard
// request signals before falling back to the cookie.
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.enabled {
			// If consent is disabled, allow everything.
			state := &State{Decided: true, Categories: make(map[string]bool)}
			for _, cat := range s.categories {
				state.Categories[cat.Key] = true
			}
			c.Set(ContextKey, state)
			c.Next()
			return
		}

		// Check for automated-tool bypass signals before reading the cookie.
		if mode, ok := s.detectBypass(c); ok {
			state := s.buildBypassState(mode)
			s.writeBypassCookie(c, state)
			c.Set(ContextKey, state)
			c.Next()
			return
		}

		state := s.readCookie(c)
		c.Set(ContextKey, state)
		c.Next()
	}
}

// detectBypass inspects the request for known automated-tool signals and
// returns the bypass mode ("accept" or "reject") if one is found.
//
// Priority order:
//  1. Sec-Purpose: prefetch  (Chromium/Lighthouse standard)
//  2. X-Purpose: prefetch    (legacy Lighthouse / crawlers)
//  3. ?consent=accept or ?consent=reject  (explicit query parameter)
//
// If the consent cookie is already decided the bypass is skipped so real
// users are never affected.
func (s *Service) detectBypass(c *gin.Context) (string, bool) {
	// Skip if the user already has a valid consent decision.
	if existing := s.readCookie(c); existing.Decided {
		return "", false
	}

	// 1. Sec-Purpose: prefetch (Chromium/Lighthouse)
	if strings.EqualFold(c.GetHeader("Sec-Purpose"), "prefetch") {
		return bypassModeAccept, true
	}

	// 2. X-Purpose: prefetch (legacy crawlers)
	if strings.EqualFold(c.GetHeader("X-Purpose"), "prefetch") {
		return bypassModeAccept, true
	}

	// 3. Explicit query parameter: ?consent=accept or ?consent=reject
	if param := c.Query("consent"); param != "" {
		switch strings.ToLower(param) {
		case bypassModeAccept:
			return bypassModeAccept, true
		case bypassModeReject:
			return bypassModeReject, true
		}
	}

	return "", false
}

// buildBypassState constructs a decided consent State for the given bypass
// mode without persisting anything to disk or a database.
//
//   - accept: all categories enabled
//   - reject: only required categories enabled
func (s *Service) buildBypassState(mode string) *State {
	state := &State{
		Decided:    true,
		Bypassed:   true,
		Categories: make(map[string]bool, len(s.categories)),
	}
	for _, cat := range s.categories {
		switch mode {
		case bypassModeAccept:
			state.Categories[cat.Key] = true
		default: // bypassModeReject
			state.Categories[cat.Key] = cat.Required
		}
	}
	return state
}

// writeBypassCookie writes the bypass consent state as a response cookie so
// that subsequent requests from the same automated tool are handled without
// a bypass signal. Cookie attributes match the regular consent cookie but
// omit Secure so local (HTTP) audit tools work out of the box.
func (s *Service) writeBypassCookie(c *gin.Context, state *State) {
	value, err := json.Marshal(state)
	if err != nil {
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieName, string(value), 365*24*3600, "/", "", false, false)
}

// readCookie parses the consent cookie into a State.
func (s *Service) readCookie(c *gin.Context) *State {
	cookie, err := c.Cookie(CookieName)
	if err != nil || cookie == "" {
		return &State{Decided: false, Categories: make(map[string]bool)}
	}

	var state State
	if err := json.Unmarshal([]byte(cookie), &state); err != nil {
		return &State{Decided: false, Categories: make(map[string]bool)}
	}

	// Enforce required categories
	for _, cat := range s.categories {
		if cat.Required {
			state.Categories[cat.Key] = true
		}
	}

	return &state
}

// Categories returns all configured categories with localized names.
func (s *Service) Categories(lang i18n.Language) []Category {
	texts, ok := s.i18nTexts[lang]
	if !ok {
		return s.categories
	}

	result := make([]Category, len(s.categories))
	copy(result, s.categories)
	for idx := range result {
		if name, ok := texts.Categories[result[idx].Key]; ok {
			result[idx].Name = name
		}
	}
	return result
}

// Texts returns the localized consent banner texts.
func (s *Service) Texts(lang i18n.Language) config.ConsentI18nConfig {
	if texts, ok := s.i18nTexts[lang]; ok {
		return texts
	}
	return config.ConsentI18nConfig{}
}

// IsEnabled returns whether consent management is enabled.
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// StateFromContext retrieves the consent state from gin context.
func StateFromContext(c *gin.Context) *State {
	state, ok := c.Get(ContextKey)
	if !ok {
		return &State{Decided: false, Categories: make(map[string]bool)}
	}
	return state.(*State)
}
