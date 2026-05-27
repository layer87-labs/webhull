package forms

import "time"

// ContactRequest represents a submitted contact form.
// Fields is a generic map of all form values keyed by field name.
// Website is the honeypot field — must be empty.
type ContactRequest struct {
	Fields  map[string]string `json:"fields"`
	Website string            `json:"website"`
}

// ContactResponse is returned after form submission.
type ContactResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ContactID string `json:"contactId,omitempty"` // unique ID for tracking
}

// FormSubmission is the internal record of a contact form submission.
type FormSubmission struct {
	Request   ContactRequest
	IP        string
	UserAgent string
	Language  string
	Timestamp time.Time
}

// Mailer is the interface for sending emails.
// Implementations: SMTPMailer, etc.
type Mailer interface {
	// Send sends a plain-text email.
	Send(to, subject, body string) error

	// SendHTML sends an HTML-only email.
	SendHTML(to, subject, htmlBody string) error

	// SendMultipart sends a multipart/alternative email with both a plain-text
	// and an HTML part. Mail clients that can render HTML will prefer the HTML
	// part; others fall back to the plain-text part.
	SendMultipart(to, subject, plainBody, htmlBody string) error
}
