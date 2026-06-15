package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	graphAPIBase   = "https://graph.instagram.com/v25.0"
	graphFBBase    = "https://graph.facebook.com/v25.0"
	requestTimeout = 15 * time.Second
)

// Client handles HTTP calls to the Instagram Graph API.
type Client struct {
	httpClient  *http.Client
	accessToken string
	userID      string
}

// NewClient creates a new Instagram API client.
func NewClient(accessToken, userID string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		accessToken: accessToken,
		userID:      userID,
	}
}

// --- Response types (internal — not exported) ---

type igMediaResponse struct {
	Data   []igMedia `json:"data"`
	Paging struct {
		Cursors struct {
			Before string `json:"before"`
			After  string `json:"after"`
		} `json:"cursors"`
		Next string `json:"next"`
	} `json:"paging"`
}

type igMedia struct {
	ID               string `json:"id"`
	Caption          string `json:"caption,omitempty"`
	MediaType        string `json:"media_type,omitempty"`
	MediaURL         string `json:"media_url,omitempty"`
	ThumbnailURL     string `json:"thumbnail_url,omitempty"`
	Permalink        string `json:"permalink,omitempty"`
	Timestamp        string `json:"timestamp,omitempty"`
	Username         string `json:"username,omitempty"`
	LikeCount        int    `json:"like_count,omitempty"`
	CommentsCount    int    `json:"comments_count,omitempty"`
	MediaProductType string `json:"media_product_type,omitempty"`
	Children         struct {
		Data []igChildMedia `json:"data,omitempty"`
	} `json:"children,omitempty"`
}

type igChildMedia struct {
	ID        string `json:"id"`
	MediaURL  string `json:"media_url,omitempty"`
	MediaType string `json:"media_type,omitempty"`
}

// tokenExchangeResponse is the response from the token exchange endpoint.
type tokenExchangeResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

// FetchMedia retrieves media from the IG User Media edge.
//
//	fields: comma-separated list of fields to request
//	limit:  max results per page (IG default/max is platform-controlled)
//	after:  pagination cursor (empty for first page)
func (c *Client) FetchMedia(ctx context.Context, fields string, limit int, after string) (*igMediaResponse, error) {
	base := fmt.Sprintf("%s/%s/media", graphAPIBase, c.userID)

	params := url.Values{}
	params.Set("access_token", c.accessToken)
	params.Set("fields", fields)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if after != "" {
		params.Set("after", after)
	}

	reqURL := base + "?" + params.Encode()

	resp, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("fetch media: %w", err)
	}

	var result igMediaResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal media response: %w", err)
	}

	return &result, nil
}

// FetchMediaByID retrieves a single media object by its Instagram ID.
// Used by the "manual" selection mode.
func (c *Client) FetchMediaByID(ctx context.Context, mediaID, fields string) (*igMedia, error) {
	reqURL := fmt.Sprintf("%s/%s?fields=%s&access_token=%s",
		graphAPIBase, mediaID, url.QueryEscape(fields), url.QueryEscape(c.accessToken))

	resp, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("fetch media by id %s: %w", mediaID, err)
	}

	var media igMedia
	if err := json.Unmarshal(resp, &media); err != nil {
		return nil, fmt.Errorf("unmarshal media %s: %w", mediaID, err)
	}

	return &media, nil
}

// RefreshToken exchanges a short-lived or expiring long-lived token for a new one.
// Endpoint: GET /refresh_access_token?grant_type=ig_refresh_token&access_token=...
// Also uses the standard exchange endpoint for initial long-lived token creation.
func (c *Client) RefreshToken(ctx context.Context, appSecret string) (string, error) {
	reqURL := fmt.Sprintf("%s/refresh_access_token", graphAPIBase)

	params := url.Values{}
	params.Set("grant_type", "ig_refresh_token")
	params.Set("access_token", c.accessToken)

	reqURL += "?" + params.Encode()

	// Append app_secret as optional param for Meta app-based exchange
	if appSecret != "" {
		reqURL += "&client_secret=" + url.QueryEscape(appSecret)
	}

	resp, err := c.doGet(ctx, reqURL)
	if err != nil {
		return "", fmt.Errorf("refresh token: %w", err)
	}

	var result tokenExchangeResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("unmarshal token refresh response: %w", err)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("token refresh returned empty access token")
	}

	return result.AccessToken, nil
}

// Ping checks connectivity to the Instagram Graph API and validates the token.
// Returns nil on success (HTTP 200), error otherwise.
func (c *Client) Ping(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/%s?fields=id,username&access_token=%s",
		graphAPIBase, c.userID, url.QueryEscape(c.accessToken))

	_, err := c.doGet(ctx, reqURL)
	return err
}

// mediaFields returns the standard set of fields requested from the API.
func mediaFields() string {
	const fields = "id,caption,media_type,media_url,thumbnail_url,permalink,timestamp,username,like_count,comments_count,media_product_type,children{id,media_url,media_type}"
	return fields
}

// doGet performs an HTTP GET and returns the raw body.
// Returns an error on non-2xx status codes.
func (c *Client) doGet(ctx context.Context, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	return body, nil
}

// APIError represents an Instagram Graph API error response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("instagram API error %d: %s", e.StatusCode, e.Body)
}
