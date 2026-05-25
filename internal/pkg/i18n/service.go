package i18n

import (
	"context"
	"strings"
)

type contextKey struct{}

// Service handles language detection and resolution.
type Service struct {
	defaultLang Language
	supported   []Language
}

// NewService creates a new i18n service.
func NewService(defaultLang string, languages []string) *Service {
	supported := make([]Language, len(languages))
	for i, l := range languages {
		supported[i] = Language(l)
	}

	return &Service{
		defaultLang: Language(defaultLang),
		supported:   supported,
	}
}

// DetectFromHeader parses the Accept-Language header and returns the best match.
func (s *Service) DetectFromHeader(acceptLanguage string) Language {
	if acceptLanguage == "" {
		return s.defaultLang
	}

	// Parse Accept-Language header (e.g., "de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7")
	parts := strings.Split(acceptLanguage, ",")
	for _, part := range parts {
		// Strip quality factor
		lang := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		// Extract primary language tag (e.g., "de-DE" → "de")
		primary := Language(strings.SplitN(lang, "-", 2)[0])

		if s.IsSupported(primary) {
			return primary
		}
	}

	return s.defaultLang
}

// DetectFromCookie reads the language preference from a cookie value.
func (s *Service) DetectFromCookie(cookieValue string) (Language, bool) {
	lang := Language(cookieValue)
	if s.IsSupported(lang) {
		return lang, true
	}
	return s.defaultLang, false
}

// IsSupported checks if a language is in the supported list.
func (s *Service) IsSupported(lang Language) bool {
	for _, l := range s.supported {
		if l == lang {
			return true
		}
	}
	return false
}

// Default returns the default language.
func (s *Service) Default() Language {
	return s.defaultLang
}

// Supported returns all supported languages.
func (s *Service) Supported() []Language {
	return s.supported
}

// WithLanguage adds the language context to a context.Context.
func WithLanguage(ctx context.Context, lc *LanguageContext) context.Context {
	return context.WithValue(ctx, contextKey{}, lc)
}

// FromContext retrieves the language context from a context.Context.
func FromContext(ctx context.Context) *LanguageContext {
	lc, ok := ctx.Value(contextKey{}).(*LanguageContext)
	if !ok {
		return nil
	}
	return lc
}
