package i18n

// Language represents a supported language code.
type Language string

const (
	LangDE Language = "de"
	LangEN Language = "en"
)

// String returns the language code as string.
func (l Language) String() string {
	return string(l)
}

// IsValid checks if the language code is in the supported list.
func (l Language) IsValid(supported []string) bool {
	for _, s := range supported {
		if string(l) == s {
			return true
		}
	}
	return false
}

// LanguageContext holds resolved language information for a request.
type LanguageContext struct {
	// Current is the resolved language for this request.
	Current Language

	// Available lists all supported languages.
	Available []Language

	// Alternates maps language → slug for the current page (for hreflang and language switcher).
	Alternates map[Language]string
}
