package security

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimiter implements in-memory rate limiting with automatic cleanup.
type RateLimiter struct {
	requests    map[string][]time.Time
	mutex       sync.RWMutex
	limit       int
	window      time.Duration
	cleanupTick time.Duration
	logger      *zap.Logger
}

// RateLimitConfig holds rate limiter parameters.
type RateLimitConfig struct {
	Limit       int
	Window      time.Duration
	CleanupTick time.Duration
}

// Preset configurations.
var (
	RateLimitContact = RateLimitConfig{
		Limit: 3, Window: 15 * time.Minute, CleanupTick: 5 * time.Minute,
	}
	RateLimitAPI = RateLimitConfig{
		Limit: 100, Window: time.Hour, CleanupTick: 10 * time.Minute,
	}
	RateLimitStrict = RateLimitConfig{
		Limit: 1, Window: 5 * time.Minute, CleanupTick: 2 * time.Minute,
	}
)

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(cfg RateLimitConfig, logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		limit:       cfg.Limit,
		window:      cfg.Window,
		cleanupTick: cfg.CleanupTick,
		logger:      logger,
	}

	if rl.cleanupTick > 0 {
		go rl.startCleanup()
	}

	return rl
}

// IsAllowed checks if a request from the identifier is within limits.
func (rl *RateLimiter) IsAllowed(identifier string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	rl.cleanupIdentifier(identifier, now)

	if len(rl.requests[identifier]) >= rl.limit {
		return false
	}

	rl.requests[identifier] = append(rl.requests[identifier], now)
	return true
}

// GetRemainingRequests returns how many requests are left for the identifier.
func (rl *RateLimiter) GetRemainingRequests(identifier string) int {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	now := time.Now()
	valid := 0
	if timestamps, exists := rl.requests[identifier]; exists {
		for _, t := range timestamps {
			if now.Sub(t) < rl.window {
				valid++
			}
		}
	}

	remaining := rl.limit - valid
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetResetTime returns when the rate limit resets for the identifier.
func (rl *RateLimiter) GetResetTime(identifier string) time.Time {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	if timestamps, exists := rl.requests[identifier]; exists && len(timestamps) > 0 {
		return timestamps[0].Add(rl.window)
	}
	return time.Now()
}

// Middleware creates a Gin middleware for rate limiting.
func (rl *RateLimiter) Middleware(identifierFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := identifierFunc(c)

		if !rl.IsAllowed(id) {
			resetTime := rl.GetResetTime(id)

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))

			c.JSON(429, gin.H{
				"error":   "rate limit exceeded",
				"message": fmt.Sprintf("try again in %v", time.Until(resetTime).Truncate(time.Second)),
			})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", rl.GetRemainingRequests(id)))

		c.Next()
	}
}

// IdentifierByIP extracts client IP as rate limit identifier.
func IdentifierByIP(c *gin.Context) string {
	return c.ClientIP()
}

func (rl *RateLimiter) cleanupIdentifier(identifier string, now time.Time) {
	timestamps, exists := rl.requests[identifier]
	if !exists {
		return
	}

	valid := timestamps[:0]
	for _, t := range timestamps {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}

	if len(valid) == 0 {
		delete(rl.requests, identifier)
	} else {
		rl.requests[identifier] = valid
	}
}

func (rl *RateLimiter) startCleanup() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		for id := range rl.requests {
			rl.cleanupIdentifier(id, now)
		}
		rl.mutex.Unlock()
	}
}
