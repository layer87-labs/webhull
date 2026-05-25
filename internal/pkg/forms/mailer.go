package forms

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPMailer implements Mailer via SMTP.
type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
	useTLS   bool
}

// NewSMTPMailer creates a new SMTP mailer from config.
func NewSMTPMailer(host string, port int, username, password, from, fromName string, useTLS bool) *SMTPMailer {
	return &SMTPMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		fromName: fromName,
		useTLS:   useTLS,
	}
}

// Send sends a plain-text email.
func (m *SMTPMailer) Send(to, subject, body string) error {
	msg := m.buildMessage(to, subject, body, "text/plain")
	return m.send(to, msg)
}

// SendHTML sends an HTML email.
func (m *SMTPMailer) SendHTML(to, subject, htmlBody string) error {
	msg := m.buildMessage(to, subject, htmlBody, "text/html")
	return m.send(to, msg)
}

func (m *SMTPMailer) buildMessage(to, subject, body, contentType string) string {
	var msg strings.Builder
	fromHeader := m.from
	if m.fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", m.fromName, m.from)
	}

	msg.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", sanitizeHeader(to)))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", sanitizeHeader(subject)))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"utf-8\"\r\n", contentType))
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return msg.String()
}

// sanitizeHeader removes CR and LF characters from an email header value
// to prevent header injection attacks.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func (m *SMTPMailer) send(to, msg string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	if m.useTLS {
		return m.sendTLS(addr, auth, to, msg)
	}

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}

func (m *SMTPMailer) sendTLS(addr string, auth smtp.Auth, to, msg string) error {
	tlsConfig := &tls.Config{
		ServerName: m.host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}
	if err = client.Mail(m.from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	// lgtm[go/email-injection] — body is intentionally user-supplied; headers are sanitized above
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}

	return client.Quit()
}
