package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// CollectorProvider forwards events to a custom analytics backend.
type CollectorProvider struct {
	endpoint string
	client   *http.Client
	logger   *zap.Logger
}

// NewCollectorProvider creates a custom analytics collector provider.
func NewCollectorProvider(endpoint string, logger *zap.Logger) *CollectorProvider {
	return &CollectorProvider{
		endpoint: strings.TrimRight(endpoint, "/"),
		client:   &http.Client{},
		logger:   logger,
	}
}

// Name returns the provider identifier.
func (c *CollectorProvider) Name() string {
	return "collector"
}

// TrackEvent forwards an event to the custom analytics backend.
func (c *CollectorProvider) TrackEvent(ctx context.Context, event Event, ip, userAgent, acceptLang string) error {
	payload := map[string]interface{}{
		"type":       event.Type,
		"url":        event.URL,
		"properties": event.Properties,
		"ip":         ip,
		"userAgent":  userAgent,
		"acceptLang": acceptLang,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal collector event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/events", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create collector request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("collector request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("collector returned status %d", resp.StatusCode)
	}

	return nil
}

// Close cleans up resources.
func (c *CollectorProvider) Close() error {
	return nil
}
