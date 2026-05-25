package forms

import (
	"strings"
	"testing"
)

func TestSpamFilter_CleanMessage(t *testing.T) {
	sf := NewSpamFilter()
	err := sf.ValidateContent("John", "Inquiry", "Hello, I am interested in your consulting services.")
	if err != nil {
		t.Errorf("clean message flagged as spam: %v", err)
	}
}

func TestSpamFilter_SpamKeywords(t *testing.T) {
	sf := NewSpamFilter()
	tests := []struct {
		name string
		msg  string
	}{
		{"viagra", "Buy cheap viagra now"},
		{"casino", "Best online casino deals"},
		{"crypto", "Invest in crypto today"},
		{"lottery", "You won the lottery"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := sf.ValidateContent("Test", "Test", tt.msg); err == nil {
				t.Errorf("expected spam for message containing %q", tt.name)
			}
		})
	}
}

func TestSpamFilter_LinkCount(t *testing.T) {
	sf := NewSpamFilter()
	// Default max links = 2
	twoLinks := "Check http://a.com and http://b.com please"
	if err := sf.ValidateContent("User", "Links", twoLinks); err != nil {
		t.Errorf("2 links should not be spam (max=2): %v", err)
	}

	threeLinks := "Visit http://a.com http://b.com http://c.com now"
	if err := sf.ValidateContent("User", "Links", threeLinks); err == nil {
		t.Error("3 links should be spam (max=2)")
	}
}

func TestSpamFilter_SetMaxLinks(t *testing.T) {
	sf := NewSpamFilter()
	sf.SetMaxLinks(0)
	if err := sf.ValidateContent("User", "Link", "Visit http://example.com"); err == nil {
		t.Error("should be spam with maxLinks=0")
	}
}

func TestSpamFilter_UpperCaseRatio(t *testing.T) {
	sf := NewSpamFilter()
	if err := sf.ValidateContent("User", "URGENT", "BUY NOW THIS IS VERY URGENT act fast"); err == nil {
		t.Error("high uppercase ratio should be spam")
	}
	if err := sf.ValidateContent("User", "Hello", "This is a normal message with some words"); err != nil {
		t.Errorf("normal case should not be spam: %v", err)
	}
}

func TestSpamFilter_WordVariety(t *testing.T) {
	sf := NewSpamFilter()
	// minWordVariety=3 — only checked for messages > 50 chars
	repeated := strings.Repeat("spam spam spam spam spam spam spam spam spam spam spam ", 2)
	if err := sf.ValidateContent("User", "Test", repeated); err == nil {
		t.Error("low word variety should be spam")
	}
}

func TestSpamFilter_BannedPhrases(t *testing.T) {
	sf := NewSpamFilter()
	tests := []struct {
		name string
		msg  string
	}{
		{"click here", "Please click here to verify"},
		{"free money", "Get your free money now"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := sf.ValidateContent("User", "Test", tt.msg); err == nil {
				t.Errorf("expected spam for banned phrase %q", tt.name)
			}
		})
	}
}

func TestSpamFilter_RepeatedCharacters(t *testing.T) {
	sf := NewSpamFilter()
	if err := sf.ValidateContent("User", "Test", "Hellooooooo there, contact me pleaseeeee"); err == nil {
		t.Error("repeated characters should be spam")
	}
	if err := sf.ValidateContent("User", "Test", "Hello there, please contact us"); err != nil {
		t.Errorf("normal text should not be spam: %v", err)
	}
}

func TestSpamFilter_AddSpamKeywords(t *testing.T) {
	sf := NewSpamFilter()
	sf.AddSpamKeywords([]string{"foobar"})
	if err := sf.ValidateContent("User", "Test", "This message contains foobar word"); err == nil {
		t.Error("custom keyword should be detected as spam")
	}
}
