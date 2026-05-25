package analytics

import "context"

// Event represents a single analytics event from the client.
type Event struct {
	// Type is the event type (e.g., "pageview", "scroll", "click", "viewport").
	Type string `json:"type" binding:"required"`

	// URL is the page URL where the event occurred.
	URL string `json:"url" binding:"required"`

	// Properties holds event-specific data.
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// PageViewEvent is a structured pageview event.
type PageViewEvent struct {
	URL      string `json:"url"`
	Referrer string `json:"referrer"`
	Title    string `json:"title"`
}

// ScrollEvent tracks how far the user scrolled.
type ScrollEvent struct {
	URL      string  `json:"url"`
	MaxDepth float64 `json:"maxDepth"` // 0.0 to 1.0
	Duration int     `json:"duration"` // time on page in seconds
}

// ClickEvent tracks user clicks.
type ClickEvent struct {
	URL      string `json:"url"`
	Selector string `json:"selector"`
	Text     string `json:"text"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
}

// Provider is the interface for analytics backends.
// Multiple providers can run in parallel.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// TrackEvent sends an event to the analytics backend.
	TrackEvent(ctx context.Context, event Event, ip, userAgent, acceptLang string) error

	// Close cleans up resources.
	Close() error
}
