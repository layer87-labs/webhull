package forms

import "testing"

func TestEmailValidator_ValidEmails(t *testing.T) {
	ev := NewEmailValidator()
	valid := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user@sub.domain.com",
	}
	for _, email := range valid {
		t.Run(email, func(t *testing.T) {
			if err := ev.ValidateEmail(email); err != nil {
				t.Errorf("valid email %q rejected: %v", email, err)
			}
		})
	}
}

func TestEmailValidator_InvalidEmails(t *testing.T) {
	ev := NewEmailValidator()
	tests := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"spaces_only", "   "},
		{"no_at", "userexample.com"},
		{"no_domain", "user@"},
		{"no_local", "@example.com"},
		{"no_tld", "user@localhost"},
		{"double_at", "user@@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ev.ValidateEmail(tt.email); err == nil {
				t.Errorf("invalid email %q should be rejected", tt.email)
			}
		})
	}
}

func TestEmailValidator_DisposableDomain(t *testing.T) {
	ev := NewEmailValidator()
	disposable := []string{
		"user@tempmail.org",
		"user@throwaway.email",
		"user@guerrillamail.com",
		"user@mailinator.com",
	}
	for _, email := range disposable {
		t.Run(email, func(t *testing.T) {
			if err := ev.ValidateEmail(email); err == nil {
				t.Errorf("disposable email %q should be blocked", email)
			}
		})
	}
}

func TestEmailValidator_AddBlockedDomains(t *testing.T) {
	ev := NewEmailValidator()
	ev.AddBlockedDomains([]string{"custom-disposable.org"})
	if err := ev.ValidateEmail("test@custom-disposable.org"); err == nil {
		t.Error("custom blocked domain should be rejected")
	}
}

func TestEmailValidator_TrimWhitespace(t *testing.T) {
	ev := NewEmailValidator()
	if err := ev.ValidateEmail("  user@example.com  "); err != nil {
		t.Errorf("whitespace-padded valid email should pass: %v", err)
	}
}
