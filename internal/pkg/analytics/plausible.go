package analytics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// PlausibleProvider implements Provider for Plausible Analytics.
type PlausibleProvider struct {
	baseURL    string
	scriptPath string
	domain     string
	client     *http.Client
	logger     *zap.Logger
}

// NewPlausibleProvider creates a Plausible analytics provider.
func NewPlausibleProvider(baseURL, scriptPath, domain string, logger *zap.Logger) *PlausibleProvider {
	return &PlausibleProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		scriptPath: scriptPath,
		domain:     domain,
		client:     &http.Client{},
		logger:     logger,
	}
}

// Name returns the provider identifier.
func (p *PlausibleProvider) Name() string {
	return "plausible"
}

// TrackEvent forwards an event to Plausible.
func (p *PlausibleProvider) TrackEvent(ctx context.Context, event Event, ip, userAgent, acceptLang string) error {
	body := fmt.Sprintf(`{"name":"%s","url":"%s","domain":"%s"}`, event.Type, event.URL, p.domain)

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/event", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create plausible request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Forwarded-For", ip)
	req.Header.Set("X-Real-IP", ip)
	if acceptLang != "" {
		req.Header.Set("Accept-Language", acceptLang)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("plausible request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("plausible returned status %d", resp.StatusCode)
	}

	return nil
}

// ProxyEvent forwards a raw event body transparently to the Plausible API.
// This preserves the exact payload format the Plausible script sends.
func (p *PlausibleProvider) ProxyEvent(ctx context.Context, body []byte, ip, userAgent, acceptLang string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/event", strings.NewReader(string(body)))
	if err != nil {
		return 0, fmt.Errorf("failed to create plausible proxy request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Forwarded-For", ip)
	req.Header.Set("X-Real-IP", ip)
	if acceptLang != "" {
		req.Header.Set("Accept-Language", acceptLang)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("plausible proxy request failed: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// Close cleans up resources.
func (p *PlausibleProvider) Close() error {
	return nil
}

// ProxyScript fetches and serves the Plausible tracking script.
func (p *PlausibleProvider) ProxyScript(ctx context.Context) ([]byte, error) {
	url := p.baseURL + p.scriptPath

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create script request: %w", err)
	}

	// Don't request compressed content from upstream; server middleware handles compression.
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plausible script: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read plausible script: %w", err)
	}

	return data, nil
}
