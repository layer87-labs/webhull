package gate

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/pkg/config"
)

const cookiePath = "/"

// sessionPayload is the signed cookie payload.
type sessionPayload struct {
	Valid bool  `json:"v"`
	TS    int64 `json:"ts"`
}

// Service validates access codes and manages signed session cookies.
type Service struct {
	codes      map[string]string // code → label (O(1) lookup)
	codeList   [][]byte          // ordered byte slices for constant-time iteration
	secret     []byte
	cookieName string
	maxAge     time.Duration
	secure     bool // true in production (Secure cookie flag)
	logger     *zap.Logger
}

// NewService creates a gate service from configuration.
// secure should be true in production (sets the Secure cookie attribute).
func NewService(cfg config.GateConfig, logger *zap.Logger, secure bool) *Service {
	codes := make(map[string]string, len(cfg.Codes))
	codeList := make([][]byte, len(cfg.Codes))
	for i, c := range cfg.Codes {
		codes[c.Code] = c.Label
		codeList[i] = []byte(c.Code)
	}

	return &Service{
		codes:      codes,
		codeList:   codeList,
		secret:     []byte(cfg.CookieSecret),
		cookieName: cfg.CookieName,
		maxAge:     cfg.CookieMaxAge,
		secure:     secure,
		logger:     logger,
	}
}

// ValidateCode checks whether the given plaintext code is valid.
// Uses crypto/subtle.ConstantTimeCompare and always iterates all codes
// without early return to prevent timing-based attacks.
func (s *Service) ValidateCode(code string) (label string, ok bool) {
	input := []byte(code)
	foundLabel := ""
	found := 0 // 0 or 1, no bool to avoid branch prediction leaks

	for _, c := range s.codeList {
		// ConstantTimeCompare returns 1 on match, 0 otherwise.
		match := subtle.ConstantTimeCompare(input, c)
		if match == 1 {
			// Record the label but do NOT break — full iteration required.
			foundLabel = s.codes[string(c)]
			found = 1
		}
	}

	return foundLabel, found == 1
}

// CreateSessionCookie signs a session payload and sets an HTTP-only cookie on the response.
// Cookie format: base64url(payload).base64url(hmac-sha256-signature)
func (s *Service) CreateSessionCookie(c *gin.Context) {
	payload := sessionPayload{Valid: true, TS: time.Now().Unix()}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	sig := s.sign([]byte(payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	cookieValue := payloadB64 + "." + sigB64
	maxAgeSec := int(s.maxAge.Seconds())

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(s.cookieName, cookieValue, maxAgeSec, cookiePath, "", s.secure, true)
}

// IsAuthenticated checks whether the request carries a valid, non-expired session cookie.
// Returns false on any error — no panic, no error logging (debug only).
func (s *Service) IsAuthenticated(c *gin.Context) bool {
	cookie, err := c.Cookie(s.cookieName)
	if err != nil || cookie == "" {
		return false
	}

	parts := strings.SplitN(cookie, ".", 2)
	if len(parts) != 2 {
		s.logger.Debug("gate: malformed cookie (missing separator)")
		return false
	}
	payloadB64, sigB64 := parts[0], parts[1]

	// Decode and verify HMAC signature first.
	// Compare the canonical base64 representation of the expected signature
	// against the raw sigB64 string — this prevents false positives caused by
	// non-canonical base64 characters that decode to the same bytes (the last
	// character of a 43-char RawURL base64 string only uses 4 of 6 bits, so
	// multiple characters can decode identically).
	expectedSig := s.sign([]byte(payloadB64))
	expectedSigB64 := base64.RawURLEncoding.EncodeToString(expectedSig)
	if subtle.ConstantTimeCompare([]byte(sigB64), []byte(expectedSigB64)) != 1 {
		s.logger.Debug("gate: signature mismatch")
		return false
	}

	// Decode payload.
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		s.logger.Debug("gate: invalid payload encoding", zap.Error(err))
		return false
	}
	var payload sessionPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		s.logger.Debug("gate: invalid payload JSON", zap.Error(err))
		return false
	}

	if !payload.Valid {
		return false
	}

	// Check expiry: issued-at + maxAge must be in the future.
	issuedAt := time.Unix(payload.TS, 0)
	if time.Since(issuedAt) > s.maxAge {
		s.logger.Debug("gate: session expired")
		return false
	}

	return true
}

// sign computes HMAC-SHA256 of data using the service secret.
func (s *Service) sign(data []byte) []byte {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(data)
	return mac.Sum(nil)
}
