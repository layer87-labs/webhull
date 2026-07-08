package i18n

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// CookieName is the cookie key used to store language preference.
	CookieName = "lang"

	// ContextKey is the gin context key for the resolved language.
	ContextKey = "i18n"
)

// Middleware creates a Gin middleware that resolves the current language
// and stores it in the request context. It also updates the language cookie.
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := s.resolve(c)

		// Set/refresh cookie (30 days, secure, httponly)
		c.SetCookie(CookieName, lang.String(), 30*24*3600, "/", "", true, false)

		// Store in gin context for handlers
		lc := &LanguageContext{
			Current:   lang,
			Available: s.Supported(),
		}
		c.Set(ContextKey, lc)

		c.Next()
	}
}

// resolve determines the language for the current request.
// Priority: 1. ?lang query param → 2. Cookie → 3. Accept-Language header → 4. Default
func (s *Service) resolve(c *gin.Context) Language {
	// 1. ?lang query param — used by language switcher in single-page mode
	if q := c.Query("lang"); q != "" {
		if lang, ok := s.DetectFromCookie(q); ok {
			return lang
		}
	}

	// 2. Cookie
	if cookie, err := c.Cookie(CookieName); err == nil {
		if lang, ok := s.DetectFromCookie(cookie); ok {
			return lang
		}
	}

	// 3. Accept-Language header
	return s.DetectFromHeader(c.GetHeader("Accept-Language"))
}

// RootRedirect handles GET / by detecting language and redirecting
// to the appropriate start page slug.
// startSlugs maps Language → slug (e.g., "de" → "/start", "en" → "/home").
func (s *Service) RootRedirect(startSlugs map[Language]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := s.resolve(c)

		slug, ok := startSlugs[lang]
		if !ok {
			slug = startSlugs[s.defaultLang]
		}

		// Set cookie before redirect
		c.SetCookie(CookieName, lang.String(), 30*24*3600, "/", "", true, false)
		c.Redirect(http.StatusMovedPermanently, "/"+slug)
	}
}
