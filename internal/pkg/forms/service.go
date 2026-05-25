package forms

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
	"github.com/layer87-labs/webhull/internal/pkg/security"
)

// Service handles contact form submissions with validation, spam detection, and dispatch.
type Service struct {
	cfg     config.ContactConfig
	mailCfg config.MailConfig
	mailer  Mailer
	spam    *SpamFilter
	email   *EmailValidator
	limiter *security.RateLimiter
	logger  *zap.Logger
}

// NewService creates a new form service.
func NewService(
	cfg config.ContactConfig,
	mailCfg config.MailConfig,
	mailer Mailer,
	limiter *security.RateLimiter,
	logger *zap.Logger,
) *Service {
	return &Service{
		cfg:     cfg,
		mailCfg: mailCfg,
		mailer:  mailer,
		spam:    NewSpamFilter(),
		email:   NewEmailValidator(),
		limiter: limiter,
		logger:  logger,
	}
}

// Submit processes a contact form submission.
func (s *Service) Submit(ctx context.Context, req ContactRequest, ip, userAgent string, lang i18n.Language) (*ContactResponse, error) {
	// Honeypot check
	if req.Website != "" {
		s.logger.Debug("honeypot triggered", zap.String("ip", ip))
		// Return success to fool bots
		return &ContactResponse{Success: true, Message: "ok"}, nil
	}

	// Rate limit check
	if !s.limiter.IsAllowed(ip) {
		return &ContactResponse{Success: false, Message: "too many requests"}, fmt.Errorf("rate limit exceeded")
	}

	// Email validation
	if err := s.email.ValidateEmail(req.Email); err != nil {
		return &ContactResponse{Success: false, Message: err.Error()}, nil
	}

	// Spam check
	if err := s.spam.ValidateContent(req.Name, req.Subject, req.Message); err != nil {
		s.logger.Info("spam detected", zap.String("ip", ip), zap.Error(err))
		return &ContactResponse{Success: false, Message: err.Error()}, nil
	}

	// Generate unique contact ID
	contactID := generateContactID()

	// Log submission immediately
	s.logger.Info("contact form submitted",
		zap.String("contactID", contactID),
		zap.String("name", req.Name),
		zap.String("email", req.Email),
		zap.String("ip", ip),
		zap.String("lang", lang.String()),
		zap.Time("timestamp", time.Now()))

	// Send emails asynchronously to avoid blocking the HTTP response
	go func() {
		// Send notification to site owner
		for _, recipient := range s.cfg.Recipients {
			subject := fmt.Sprintf("[%s] %s - Ref: %s", req.Name, req.Subject, contactID)
			body := fmt.Sprintf("Contact ID: %s\nFrom: %s <%s>\nSubject: %s\n\n%s", contactID, req.Name, req.Email, req.Subject, req.Message)

			if err := s.mailer.Send(recipient, subject, body); err != nil {
				s.logger.Error("failed to send contact notification",
					zap.String("contactID", contactID),
					zap.String("recipient", recipient),
					zap.Error(err))
			}
		}

		// Send auto-reply to contact with original message
		if tmpl, ok := s.mailCfg.Templates[lang.String()]; ok {
			// Prepare template data with HTML-escaped original message
			body := s.renderAutoReply(tmpl.Body, contactID, req)
			subject := fmt.Sprintf("%s - Ref: %s", tmpl.Subject, contactID)

			if err := s.mailer.SendHTML(req.Email, subject, body); err != nil {
				s.logger.Warn("failed to send auto-reply",
					zap.String("contactID", contactID),
					zap.String("email", req.Email),
					zap.Error(err))
			}
		}
	}()

	return &ContactResponse{Success: true, Message: "ok", ContactID: contactID}, nil
}

// generateContactID creates a unique short ID for contact tracking.
// Format: 8 hex chars (e.g., "a3f2b8e1") - compact but unique enough.
func generateContactID() string {
	b := make([]byte, 4) // 4 bytes = 8 hex chars
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%08x", time.Now().Unix())
	}
	return hex.EncodeToString(b)
}

// renderAutoReply injects contact ID and original message into the template.
// Template placeholders: {{CONTACT_ID}}, {{NAME}}, {{SUBJECT}}, {{MESSAGE}}
func (s *Service) renderAutoReply(template string, contactID string, req ContactRequest) string {
	// HTML-escape user input to prevent XSS, and convert newlines to <br>
	escapedMessage := html.EscapeString(req.Message)
	escapedMessage = strings.ReplaceAll(escapedMessage, "\n", "<br>")

	replacements := map[string]string{
		"{{CONTACT_ID}}": html.EscapeString(contactID),
		"{{NAME}}":       html.EscapeString(req.Name),
		"{{SUBJECT}}":    html.EscapeString(req.Subject),
		"{{MESSAGE}}":    escapedMessage,
	}

	result := template
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}
