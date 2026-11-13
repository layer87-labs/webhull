package forms

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

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
	// autoReplyLimiter caps auto-replies across all clients under one key.
	autoReplyLimiter *security.RateLimiter
	logger           *zap.Logger
}

// autoReplyKey is the single identifier of the global auto-reply limiter.
const autoReplyKey = "auto-reply"

// fieldPlaceholder matches any {{FIELD}} placeholder in an auto-reply template.
var fieldPlaceholder = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)

// NewService creates a new form service.
func NewService(
	cfg config.ContactConfig,
	mailCfg config.MailConfig,
	mailer Mailer,
	limiter *security.RateLimiter,
	logger *zap.Logger,
) *Service {
	replyLimit := cfg.AutoReplyRateLimit
	if replyLimit.Requests <= 0 {
		replyLimit.Requests = config.DefaultAutoReplyRequests
	}
	if replyLimit.Window <= 0 {
		replyLimit.Window = config.DefaultAutoReplyWindow
	}

	for lang, tmpl := range mailCfg.Templates {
		for _, p := range fieldPlaceholder.FindAllString(tmpl.Body, -1) {
			if p != "{{CONTACT_ID}}" {
				logger.Warn("auto-reply template placeholder is no longer filled; only {{CONTACT_ID}} is supported",
					zap.String("lang", lang), zap.String("placeholder", p))
			}
		}
	}

	return &Service{
		cfg:     cfg,
		mailCfg: mailCfg,
		mailer:  mailer,
		spam:    NewSpamFilter(),
		email:   NewEmailValidator(),
		limiter: limiter,
		autoReplyLimiter: security.NewRateLimiter(security.RateLimitConfig{
			Limit:       replyLimit.Requests,
			Window:      replyLimit.Window,
			CleanupTick: replyLimit.Window,
		}, logger),
		logger: logger,
	}
}

// Submit processes a contact form submission.
// fieldDefs defines the expected fields (order, labels, types, required).
// When fieldDefs is empty the submission is accepted as-is (fallback).
func (s *Service) Submit(ctx context.Context, req ContactRequest, ip, userAgent string, lang i18n.Language, fieldDefs []config.FieldConfig) (*ContactResponse, error) {
	// Honeypot check
	if req.Website != "" {
		s.logger.Debug("honeypot triggered", zap.String("ip", ip))
		return &ContactResponse{Success: true, Message: "ok"}, nil
	}

	// Allowlist: strip any fields not declared in fieldDefs to prevent
	// frontend injection of unexpected keys (e.g. bcc, template, admin).
	if len(fieldDefs) > 0 {
		allowed := make(map[string]string, len(fieldDefs))
		for _, def := range fieldDefs {
			if v, ok := req.Fields[def.Name]; ok {
				allowed[def.Name] = v
			}
		}
		req.Fields = allowed
	} else {
		for name, v := range req.Fields {
			if utf8.RuneCountInString(v) > config.DefaultTextareaMaxLength {
				return &ContactResponse{Success: false, Message: "field too long: " + name}, nil
			}
		}
	}

	// Rate limit check
	if !s.limiter.IsAllowed(ip) {
		return &ContactResponse{Success: false, Message: "too many requests"}, fmt.Errorf("rate limit exceeded")
	}

	// Validate required fields and lengths, collect email value
	var emailVal string
	for _, def := range fieldDefs {
		val := strings.TrimSpace(req.Fields[def.Name])
		if def.Required && val == "" {
			return &ContactResponse{Success: false, Message: "invalid request"}, nil
		}
		if utf8.RuneCountInString(val) > def.EffectiveMaxLength() {
			return &ContactResponse{Success: false, Message: "field too long: " + def.Name}, nil
		}
		if def.Type == "email" && strings.ContainsAny(req.Fields[def.Name], "\r\n") {
			return &ContactResponse{Success: false, Message: "invalid email address"}, nil
		}
		if def.Type == "email" && emailVal == "" {
			emailVal = val
		}
	}

	// Email validation (first email-type field)
	if emailVal == "" {
		if strings.ContainsAny(req.Fields["email"], "\r\n") {
			return &ContactResponse{Success: false, Message: "invalid email address"}, nil
		}
		emailVal = strings.TrimSpace(req.Fields["email"])
	}
	if err := s.email.ValidateEmail(emailVal); err != nil {
		return &ContactResponse{Success: false, Message: err.Error()}, nil
	}

	// Spam check: combine all field values
	var combined strings.Builder
	for _, v := range req.Fields {
		combined.WriteString(v)
		combined.WriteString(" ")
	}
	if err := s.spam.ValidateContent("", "", combined.String()); err != nil {
		s.logger.Info("spam detected", zap.String("ip", ip), zap.Error(err))
		return &ContactResponse{Success: false, Message: err.Error()}, nil
	}

	contactID := generateContactID()

	// Derive display name for logging (conventional "name" field)
	nameVal := strings.TrimSpace(req.Fields["name"])

	// Sanitize user-controlled values before logging to prevent log forging.
	safeNameVal := strings.NewReplacer("\n", "", "\r", "").Replace(nameVal)
	safeEmailVal := strings.NewReplacer("\n", "", "\r", "").Replace(emailVal)

	s.logger.Info("contact form submitted",
		zap.String("contactID", contactID),
		zap.String("name", safeNameVal),
		zap.String("email", safeEmailVal),
		zap.String("ip", ip),
		zap.String("lang", lang.String()),
		zap.Time("timestamp", time.Now()))

	go func() {
		// Notification email to site owner — fields in configured order
		emailSubject := s.buildNotificationSubject(req, fieldDefs, contactID)
		plainBody, htmlBody := s.buildNotificationBodies(req, fieldDefs, contactID)

		for _, recipient := range s.cfg.Recipients {
			if err := s.mailer.SendMultipart(recipient, emailSubject, plainBody, htmlBody); err != nil {
				s.logger.Error("failed to send contact notification",
					zap.String("contactID", contactID),
					zap.String("recipient", recipient),
					zap.Error(err))
			}
		}

		// Auto-reply to the validated address. It goes to an address anyone
		// can enter, so it carries a fixed text and the reference only.
		if tmpl, ok := s.mailCfg.Templates[lang.String()]; ok {
			if !s.autoReplyLimiter.IsAllowed(autoReplyKey) {
				s.logger.Warn("auto-reply skipped: contact.autoReplyRateLimit reached",
					zap.String("contactID", contactID))
				return
			}
			body := renderAutoReply(tmpl.Body, contactID)
			subject := fmt.Sprintf("%s - Ref: %s", tmpl.Subject, contactID)
			if err := s.mailer.SendHTML(emailVal, subject, body); err != nil {
				s.logger.Warn("failed to send auto-reply",
					zap.String("contactID", contactID),
					zap.String("email", safeEmailVal),
					zap.Error(err))
			}
		}
	}()

	return &ContactResponse{Success: true, Message: "ok", ContactID: contactID}, nil
}

// buildNotificationSubject derives the email subject for the site owner notification.
// Uses "name" + "subject" fields if present, otherwise first non-email non-name field.
func (s *Service) buildNotificationSubject(req ContactRequest, defs []config.FieldConfig, contactID string) string {
	name := strings.TrimSpace(req.Fields["name"])
	subj := strings.TrimSpace(req.Fields["subject"])
	if subj == "" {
		for _, def := range defs {
			if def.Name != "name" && def.Type != "email" {
				subj = strings.TrimSpace(req.Fields[def.Name])
				if len(subj) > 60 {
					subj = subj[:60] + "…"
				}
				break
			}
		}
	}
	return fmt.Sprintf("[%s] %s - Ref: %s", name, subj, contactID)
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

// renderAutoReply fills {{CONTACT_ID}} and removes every other placeholder.
// Submitted field values are never copied into the auto-reply.
func renderAutoReply(template string, contactID string) string {
	body := strings.ReplaceAll(template, "{{CONTACT_ID}}", html.EscapeString(contactID))
	return fieldPlaceholder.ReplaceAllString(body, "")
}

// buildNotificationBodies renders the site-owner notification as both plain text and HTML.
func (s *Service) buildNotificationBodies(req ContactRequest, defs []config.FieldConfig, contactID string) (plain, htmlBody string) {
	// Plain text
	var pt strings.Builder
	pt.WriteString("NEUE KONTAKTANFRAGE\n")
	pt.WriteString(strings.Repeat("-", 40) + "\n")
	for _, def := range defs {
		pt.WriteString(fmt.Sprintf("%-20s %s\n", def.Label+":", req.Fields[def.Name]))
	}
	pt.WriteString(strings.Repeat("-", 40) + "\n")
	pt.WriteString(fmt.Sprintf("Referenz: %s\n", contactID))

	// HTML
	var rows strings.Builder
	for _, def := range defs {
		val := html.EscapeString(req.Fields[def.Name])
		val = strings.ReplaceAll(val, "\n", "<br>")
		rows.WriteString(fmt.Sprintf(`
		<tr>
			<td style="padding:10px 16px;font-size:13px;color:#666;font-weight:600;white-space:nowrap;vertical-align:top;width:160px;border-bottom:1px solid #f0f0f0;">%s</td>
			<td style="padding:10px 16px;font-size:14px;color:#111;vertical-align:top;border-bottom:1px solid #f0f0f0;">%s</td>
		</tr>`, html.EscapeString(def.Label), val))
	}

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html lang="de">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f5f5f5;padding:32px 16px;">
  <tr><td align="center">
    <table width="600" cellpadding="0" cellspacing="0" style="background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 1px 4px rgba(0,0,0,.08);max-width:600px;width:100%%;">
      <tr>
        <td style="background:#111;padding:20px 24px;">
          <span style="color:#fff;font-size:13px;font-weight:600;letter-spacing:.05em;">NEUE KONTAKTANFRAGE</span>
        </td>
      </tr>
      <tr>
        <td style="padding:24px 16px 8px;">
          <table width="100%%" cellpadding="0" cellspacing="0">%s</table>
        </td>
      </tr>
      <tr>
        <td style="padding:16px 24px 24px;">
          <p style="margin:0;font-size:12px;color:#999;">Referenz: <code style="background:#f5f5f5;padding:2px 6px;border-radius:3px;font-family:monospace;">%s</code></p>
        </td>
      </tr>
    </table>
  </td></tr>
</table>
</body>
</html>`, rows.String(), html.EscapeString(contactID))

	return pt.String(), htmlBody
}
