package security

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestRateLimiter_IsAllowed(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Limit: 3, Window: time.Minute}, testLogger())

	for i := 0; i < 3; i++ {
		if !rl.IsAllowed("test-id") {
			t.Errorf("request %d should be allowed (limit=3)", i+1)
		}
	}

	if rl.IsAllowed("test-id") {
		t.Error("4th request should be rejected (limit=3)")
	}
}

func TestRateLimiter_DifferentIdentifiers(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Limit: 1, Window: time.Minute}, testLogger())

	if !rl.IsAllowed("user-a") {
		t.Error("user-a first request should be allowed")
	}
	if !rl.IsAllowed("user-b") {
		t.Error("user-b first request should be allowed (different identifier)")
	}
	if rl.IsAllowed("user-a") {
		t.Error("user-a second request should be rejected")
	}
}

func TestRateLimiter_GetRemainingRequests(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Limit: 5, Window: time.Minute}, testLogger())

	if r := rl.GetRemainingRequests("fresh"); r != 5 {
		t.Errorf("remaining = %d, want 5 for new identifier", r)
	}

	rl.IsAllowed("fresh")
	rl.IsAllowed("fresh")
	if r := rl.GetRemainingRequests("fresh"); r != 3 {
		t.Errorf("remaining = %d, want 3 after 2 requests", r)
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Limit: 1, Window: 50 * time.Millisecond}, testLogger())

	if !rl.IsAllowed("expire-test") {
		t.Fatal("first request should be allowed")
	}
	if rl.IsAllowed("expire-test") {
		t.Fatal("second request should be rejected")
	}

	time.Sleep(60 * time.Millisecond)

	if !rl.IsAllowed("expire-test") {
		t.Error("request after window expiry should be allowed again")
	}
}

func TestRateLimiter_Presets(t *testing.T) {
	tests := []struct {
		name  string
		cfg   RateLimitConfig
		limit int
	}{
		{"Contact", RateLimitContact, 3},
		{"API", RateLimitAPI, 100},
		{"Strict", RateLimitStrict, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.cfg, testLogger())
			for i := 0; i < tt.limit; i++ {
				if !rl.IsAllowed("preset-test") {
					t.Errorf("request %d should be allowed (limit=%d)", i+1, tt.limit)
				}
			}
			if rl.IsAllowed("preset-test") {
				t.Errorf("request %d should be rejected (limit=%d)", tt.limit+1, tt.limit)
			}
		})
	}
}

func TestRateLimiter_GetResetTime(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Limit: 1, Window: time.Minute}, testLogger())

	rl.IsAllowed("reset-test")
	resetTime := rl.GetResetTime("reset-test")
	if resetTime.IsZero() {
		t.Error("reset time should not be zero after a request")
	}
	if resetTime.Before(time.Now()) {
		t.Error("reset time should be in the future")
	}
}
