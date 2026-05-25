package forms

import "time"

// ContactRequest represents a submitted contact form.
type ContactRequest struct {
	Name    string `json:"name" binding:"required"`
	Email   string `json:"email" binding:"required"`
	Subject string `json:"subject" binding:"required"`
	Message string `json:"message" binding:"required"`

	// Honeypot field — must be empty (bot trap)
	Website string `json:"website"`
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
	// Send sends an email.
	Send(to, subject, body string) error

	// SendHTML sends an HTML email.
	SendHTML(to, subject, htmlBody string) error
}
