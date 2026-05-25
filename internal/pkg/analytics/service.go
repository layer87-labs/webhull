package analytics

import (
	"context"

	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/pkg/consent"
)

// Service dispatches analytics events to all configured providers.
// It respects the user's consent state.
type Service struct {
	providers []Provider
	logger    *zap.Logger
}

// NewService creates a new analytics service with the given providers.
func NewService(logger *zap.Logger, providers ...Provider) *Service {
	return &Service{
		providers: providers,
		logger:    logger,
	}
}

// Track dispatches an event to all providers if analytics consent is given.
func (s *Service) Track(ctx context.Context, consentState *consent.State, event Event, ip, userAgent, acceptLang string) {
	// Check analytics consent
	if consentState != nil && !consentState.IsAllowed("analytics") {
		return
	}

	for _, provider := range s.providers {
		if err := provider.TrackEvent(ctx, event, ip, userAgent, acceptLang); err != nil {
			s.logger.Warn("analytics provider error",
				zap.String("provider", provider.Name()),
				zap.Error(err))
		}
	}
}

// TrackServerSide sends a pageview server-side when client-side tracking
// is not active (consent not given). This provides anonymous base analytics
// without requiring consent, since no cookies or personal data are stored.
// When consent IS given, the client-side JS handles tracking (deduplication).
func (s *Service) TrackServerSide(consentState *consent.State, pageURL, ip, userAgent, acceptLang string) {
	// Skip if analytics consent is given — client JS will track instead
	if consentState != nil && consentState.IsAllowed("analytics") {
		return
	}

	event := Event{
		Type: "pageview",
		URL:  pageURL,
	}

	// Fire-and-forget with background context (response already sent)
	ctx := context.Background()
	for _, provider := range s.providers {
		provider := provider
		go func() {
			if err := provider.TrackEvent(ctx, event, ip, userAgent, acceptLang); err != nil {
				s.logger.Debug("server-side tracking failed",
					zap.String("provider", provider.Name()),
					zap.Error(err))
			}
		}()
	}
}

// Close shuts down all providers.
func (s *Service) Close() {
	for _, provider := range s.providers {
		if err := provider.Close(); err != nil {
			s.logger.Warn("failed to close analytics provider",
				zap.String("provider", provider.Name()),
				zap.Error(err))
		}
	}
}

// Providers returns all registered analytics providers.
// Used by handlers that need direct access to a specific provider (e.g., Plausible proxy).
func (s *Service) Providers() []Provider {
	return s.providers
}
