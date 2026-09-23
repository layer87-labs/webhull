package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// request is one outgoing GET — the fields Source and EnrichSource share.
type request struct {
	URL     string
	Query   map[string]string
	Headers map[string]string
	Auth    *Auth
}

// fetch performs one GET against the manifest's source and returns the raw
// JSON body. Never called from a request path — only from the background
// refresh loop in instance.go.
func fetch(ctx context.Context, client *http.Client, src Source) ([]byte, error) {
	return fetchURL(ctx, client, request{URL: src.URL, Query: src.Query, Headers: src.Headers, Auth: src.Auth})
}

// fetchURL performs one GET and returns the body, verified to be valid
// JSON. The body is returned raw rather than decoded: decoding into
// interface{} loses object key order, which a keyed collection needs (see
// collectionAt). Shared by the base list fetch and the per-item enrich fetch.
func fetchURL(ctx context.Context, client *http.Client, r request) ([]byte, error) {
	u, err := url.Parse(r.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	q := u.Query()
	for k, v := range r.Query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	if r.Auth != nil && r.Auth.Basic != nil {
		req.SetBasicAuth(r.Auth.Basic.Username, r.Auth.Basic.Password)
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
	if !json.Valid(body) {
		return nil, errors.New("parse response as JSON: invalid JSON")
	}
	return body, nil
}
