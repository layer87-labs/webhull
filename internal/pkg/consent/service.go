package consent

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
)

const (
	// CookieName is the consent cookie key.
	CookieName = "consent"

	// ContextKey is the gin context key for consent state.
	ContextKey = "consent"
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
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.enabled {
			// If consent is disabled, allow everything
			state := &State{Decided: true, Categories: make(map[string]bool)}
			for _, cat := range s.categories {
				state.Categories[cat.Key] = true
			}
			c.Set(ContextKey, state)
			c.Next()
			return
		}

		state := s.readCookie(c)
		c.Set(ContextKey, state)
		c.Next()
	}
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
