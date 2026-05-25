package navigation

import "github.com/layer87-labs/webhull/internal/pkg/i18n"

// Item represents a single navigation entry.
type Item struct {
	Title    string
	URL      string
	Active   bool
	Children []Item
}

// Header holds the resolved header navigation for the current request.
type Header struct {
	Items    []Item
	Language i18n.Language
}

// Footer holds the resolved footer navigation for the current request.
type Footer struct {
	Items    []Item
	Language i18n.Language
}
