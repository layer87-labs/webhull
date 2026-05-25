package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CacheMiddleware sets Cache-Control headers based on request path.
// HTML pages use no-cache to force revalidation (ETag handled in handlers).
// Static assets use long-lived caches (they have cache-busting hashes).
func CacheMiddleware(cacheMaxAge, staticCacheMaxAge time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		switch {
		case strings.HasPrefix(path, "/static/css/"), strings.HasPrefix(path, "/static/js/"):
			setCacheHeader(c, staticCacheMaxAge, true)
		case strings.HasPrefix(path, "/static/img/"):
			setCacheHeader(c, 30*24*time.Hour, false)
		case path == "/sitemap.xml", path == "/robots.txt":
			setCacheHeader(c, 24*time.Hour, false)
		default:
			// HTML pages: always revalidate with the server.
			// Handlers set ETag for conditional 304 responses.
			c.Header("Cache-Control", "no-cache")
		}

		c.Next()
	}
}

func setCacheHeader(c *gin.Context, maxAge time.Duration, immutable bool) {
	cc := fmt.Sprintf("public, max-age=%d", int(maxAge.Seconds()))
	if immutable {
		cc += ", immutable"
	}
	c.Header("Cache-Control", cc)
}

// GzipMiddleware provides smart gzip compression excluding already-compressed formats.
func GzipMiddleware() gin.HandlerFunc {
	return gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedExtensions([]string{
			".gz", ".br", ".zip",
			".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".ico", ".svg",
			".woff", ".woff2", ".ttf", ".eot",
			".mp4", ".webm", ".ogg", ".mp3",
		}),
	)
}

// LoggingMiddleware logs slow requests and errors.
func LoggingMiddleware(logger *zap.Logger, slowThreshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
		}

		switch {
		case status >= 500:
			logger.Error("server error", fields...)
		case status >= 400:
			// Client errors — Info level, no stacktrace
			logger.Info("client error", fields...)
		case latency > slowThreshold:
			logger.Warn("slow request", fields...)
		}
	}
}
