package consent

// State represents the user's consent choices.
type State struct {
	// Decided is true if the user has made a choice (accepted/rejected).
	Decided bool `json:"decided"`

	// Categories maps category key → accepted (true/false).
	Categories map[string]bool `json:"categories"`

	// Bypassed is true when consent was resolved by the server-side bypass
	// middleware (e.g. Lighthouse, Playwright, Unlighthouse). It is a
	// transient, request-scoped flag — never serialised to the cookie.
	Bypassed bool `json:"-"`
}

// Category defines a single consent category.
type Category struct {
	Key      string
	Name     string // localized display name
	Required bool   // cannot be disabled (e.g., "necessary")
	Default  bool   // default state if user hasn't decided
}

// IsAllowed checks if a specific category has been accepted.
func (s *State) IsAllowed(category string) bool {
	if !s.Decided {
		return false
	}
	allowed, ok := s.Categories[category]
	return ok && allowed
}
