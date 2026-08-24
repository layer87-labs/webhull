package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// fetch performs one GET against the manifest's source and returns the
// parsed JSON body. Never called from a request path — only from the
// background refresh loop in instance.go.
func fetch(ctx context.Context, client *http.Client, src Source) (interface{}, error) {
	return fetchURL(ctx, client, src.URL, src.Query, src.Headers)
}

// fetchURL performs one GET against rawURL with the given query params and
// headers, and returns the parsed JSON body. Shared by the base list fetch
// (fetch, above) and the per-item enrich fetch (instance.go).
func fetchURL(ctx context.Context, client *http.Client, rawURL string, query, headers map[string]string) (interface{}, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	q := u.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // 8 MiB guard
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %s", resp.Status)
	}

	var parsed interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse response as JSON: %w", err)
	}
	return parsed, nil
}
