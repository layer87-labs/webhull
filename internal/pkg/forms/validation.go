package forms

import (
	"fmt"
	"regexp"
	"strings"
)

// EmailValidator provides robust email validation with disposable domain blocking.
type EmailValidator struct {
	blockedDomains map[string]struct{}
	emailRegex     *regexp.Regexp
}

// NewEmailValidator creates a validator with default blocked domains.
func NewEmailValidator() *EmailValidator {
	blocked := make(map[string]struct{})
	for _, d := range defaultBlockedDomains() {
		blocked[d] = struct{}{}
	}
	return &EmailValidator{
		blockedDomains: blocked,
		emailRegex:     regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`),
	}
}

// ValidateEmail validates format and checks against disposable domains.
func (v *EmailValidator) ValidateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		return fmt.Errorf("email address is required")
	}
	if !v.emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email address")
	}

	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid email address")
	}

	local, domain := parts[0], parts[1]
	if len(local) > 64 || len(domain) > 255 {
		return fmt.Errorf("invalid email address")
	}
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("invalid email address")
	}

	if _, blocked := v.blockedDomains[domain]; blocked {
		return fmt.Errorf("disposable email addresses are not allowed")
	}

	return nil
}

// AddBlockedDomains adds additional domains to the blocklist.
func (v *EmailValidator) AddBlockedDomains(domains []string) {
	for _, d := range domains {
		v.blockedDomains[strings.TrimSpace(strings.ToLower(d))] = struct{}{}
	}
}

func defaultBlockedDomains() []string {
	return []string{
		"tempmail.org", "10minutemail.com", "guerrillamail.com",
		"mailinator.com", "throwaway.email", "temp-mail.org",
		"yopmail.com", "maildrop.cc", "sharklasers.com",
		"trashmail.com", "trashmail.org", "trashmail.net",
		"mohmal.com", "mailcatch.com", "emailondeck.com",
		"dispostable.com", "spamgourmet.com",
	}
}
