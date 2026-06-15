package instagram

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Service manages Instagram feed data — fetching, caching, and token lifecycle.
type Service struct {
	cfg      FeedRequest
	cacheTTL time.Duration
	client   *Client
	logger   *zap.Logger

	mu    sync.RWMutex
	cache *cachedFeed

	// Token management
	tokenMu          sync.Mutex
	tokenRefreshedAt time.Time     // when the token was last refreshed
	stopCh           chan struct{} // signals the refresh goroutine to stop
	wg               sync.WaitGroup
}

// NewService creates a new Instagram feed service.
// It starts a background goroutine for periodic token refresh.
//
// cfg must have SelectionMode, Count, FetchMultiplier, ExcludeVideo,
// FilterMediaProductType, and ManualPostIDs populated from the user config.
func NewService(cfg FeedRequest, accessToken, userID, appSecret string, cacheTTL time.Duration, tokenRefreshDaysBefore int, logger *zap.Logger) *Service {
	client := NewClient(accessToken, userID)

	s := &Service{
		cfg:      cfg,
		cacheTTL: cacheTTL,
		client:   client,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}

	// Start background token refresh goroutine.
	// We don't have the exact expiry time from a freshly-configured token,
	// so we must infer it. The first refresh happens after a warm-up period.
	s.wg.Add(1)
	go s.tokenRefreshLoop(accessToken, appSecret, tokenRefreshDaysBefore)

	return s
}

// GetPosts returns the current feed of Instagram posts.
// Serves from cache when fresh; fetches from API when stale. Subsequent
// callers waiting for a concurrent fetch will block on the same mutex
// and receive the freshly-cached result.
//
// On API error, returns the last valid cached data (never fails the page).
// Returns nil only when there has never been a successful fetch (cold cache
// + API error on first attempt).
func (s *Service) GetPosts(ctx context.Context) []Post {
	// Fast path: cache is fresh → return immediately.
	s.mu.RLock()
	if s.cache != nil && time.Since(s.cache.FetchedAt) < s.cacheTTL {
		posts := s.cache.Posts
		s.mu.RUnlock()
		return posts
	}
	// Need to check if we should fetch — must get write lock.
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check: another goroutine may have fetched while we waited.
	if s.cache != nil && time.Since(s.cache.FetchedAt) < s.cacheTTL {
		return s.cache.Posts
	}

	// Fetch from API.
	posts, err := s.fetchAndProcess(ctx)
	if err != nil {
		s.logger.Error("instagram fetch failed, serving stale cache", zap.Error(err))
		// Return stale cache if available.
		if s.cache != nil {
			return s.cache.Posts
		}
		// Cold cache — return empty, log warning.
		s.logger.Warn("instagram feed has no cached data and API fetch failed")
		return nil
	}

	// Update cache.
	s.cache = &cachedFeed{
		Posts:     posts,
		FetchedAt: time.Now(),
	}

	return posts
}

// HasPosts returns true if there are cached posts available for rendering.
// This is used by templates to decide whether to show the feed section.
func (s *Service) HasPosts() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache != nil && len(s.cache.Posts) > 0
}

// Stop gracefully shuts down the background token refresh goroutine.
// Safe to call multiple times.
func (s *Service) Stop() {
	select {
	case <-s.stopCh:
		// Already closed.
	default:
		close(s.stopCh)
	}
	s.wg.Wait()
}

// cacheTTL is the effective cache TTL from the config.
func (r FeedRequest) cacheTTL() time.Duration {
	// cacheTTL is not directly available on FeedRequest; Service stores cfg
	// but the TTL is known at construction time only from the outer config.
	// This method is a stub; the actual TTL is stored via the outer config.
	return 0
}

// fetchAndProcess fetches media from the API and applies selection/filtering.
func (s *Service) fetchAndProcess(ctx context.Context) ([]Post, error) {
	switch s.cfg.SelectionMode {
	case ModeLatestN:
		return s.fetchLatest(ctx)
	case ModeTopEngagement:
		return s.fetchTopEngagement(ctx)
	case ModeManual:
		return s.fetchManual(ctx)
	default:
		return nil, fmt.Errorf("unknown selection mode: %s", s.cfg.SelectionMode)
	}
}

// fetchLatest fetches the most recent posts and returns up to Count.
func (s *Service) fetchLatest(ctx context.Context) ([]Post, error) {
	limit := s.cfg.Count * s.cfg.FetchMultiplier

	mediaResp, err := s.client.FetchMedia(ctx, mediaFields(), limit, "")
	if err != nil {
		return nil, err
	}

	posts := s.filterAndConvert(mediaResp.Data)

	// Trim to Count.
	if len(posts) > s.cfg.Count {
		posts = posts[:s.cfg.Count]
	}

	return posts, nil
}

// fetchTopEngagement fetches more posts than needed, then ranks by engagement.
func (s *Service) fetchTopEngagement(ctx context.Context) ([]Post, error) {
	limit := s.cfg.Count * s.cfg.FetchMultiplier

	mediaResp, err := s.client.FetchMedia(ctx, mediaFields(), limit, "")
	if err != nil {
		return nil, err
	}

	posts := s.filterAndConvert(mediaResp.Data)

	// Sort by engagement (likes + comments) descending.
	sort.Slice(posts, func(i, j int) bool {
		engI := posts[i].LikeCount + posts[i].CommentsCount
		engJ := posts[j].LikeCount + posts[j].CommentsCount
		return engI > engJ
	})

	// Trim to Count.
	if len(posts) > s.cfg.Count {
		posts = posts[:s.cfg.Count]
	}

	return posts, nil
}

// fetchManual fetches posts by their explicit IDs.
func (s *Service) fetchManual(ctx context.Context) ([]Post, error) {
	posts := make([]Post, 0, len(s.cfg.ManualPostIDs))
	fields := mediaFields()

	for _, id := range s.cfg.ManualPostIDs {
		media, err := s.client.FetchMediaByID(ctx, id, fields)
		if err != nil {
			s.logger.Error("instagram manual fetch failed for id",
				zap.String("id", id),
				zap.Error(err))
			continue // skip individual failures, continue with remaining posts
		}

		post := convertMedia(media)
		posts = append(posts, post)
	}

	// Manual mode: order as defined in config, trimmed to Count.
	if len(posts) > s.cfg.Count {
		posts = posts[:s.cfg.Count]
	}

	return posts, nil
}

// filterAndConvert filters raw IG media by type/product and converts to Post.
func (s *Service) filterAndConvert(media []igMedia) []Post {
	posts := make([]Post, 0, len(media))

	for _, m := range media {
		// Filter by media type.
		if s.cfg.ExcludeVideo && (m.MediaType == "VIDEO" || m.MediaType == "IGTV") {
			continue
		}

		// Filter by media product type.
		if len(s.cfg.FilterMediaProductType) > 0 {
			allowed := false
			for _, pt := range s.cfg.FilterMediaProductType {
				if m.MediaProductType == pt {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		posts = append(posts, convertMedia(&m))
	}

	return posts
}

// convertMedia converts a raw IG API media object to a Post.
// For CAROUSEL_ALBUM, the first child's MediaURL is used as the primary image.
func convertMedia(m *igMedia) Post {
	ts, _ := time.Parse(time.RFC3339, m.Timestamp)

	post := Post{
		ID:               m.ID,
		Caption:          m.Caption,
		MediaType:        m.MediaType,
		MediaURL:         m.MediaURL,
		ThumbnailURL:     m.ThumbnailURL,
		Permalink:        m.Permalink,
		Timestamp:        ts,
		Username:         m.Username,
		LikeCount:        m.LikeCount,
		CommentsCount:    m.CommentsCount,
		MediaProductType: m.MediaProductType,
	}

	// CAROUSEL_ALBUM: use first child image as primary MediaURL.
	if m.MediaType == "CAROUSEL_ALBUM" && len(m.Children.Data) > 0 {
		children := make([]ChildMedia, 0, len(m.Children.Data))
		for _, child := range m.Children.Data {
			children = append(children, ChildMedia{
				ID:        child.ID,
				MediaURL:  child.MediaURL,
				MediaType: child.MediaType,
			})
		}
		post.Children = children

		// Use first child's URL as the primary display image.
		if children[0].MediaURL != "" {
			post.MediaURL = children[0].MediaURL
		}
	}

	return post
}

// tokenRefreshLoop periodically refreshes the access token before it expires.
// Long-lived tokens expire after ~60 days. This goroutine refreshes daily,
// but only actually calls the API within the configured window before expiry
// (default: 7 days). The first refresh happens after a warm-up period.
func (s *Service) tokenRefreshLoop(accessToken, appSecret string, refreshDaysBefore int) {
	defer s.wg.Done()

	// Warm-up: wait 5 minutes before first check (allows server to stabilize).
	timer := time.NewTimer(5 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			if err := s.maybeRefreshToken(accessToken, appSecret, refreshDaysBefore); err != nil {
				s.logger.Error("instagram token refresh failed", zap.Error(err))
			}

			// Check daily.
			timer.Reset(24 * time.Hour)
		}
	}
}

// maybeRefreshToken refreshes the access token if it's close to expiry.
// Since we don't store the exact token expiry time, we track the last refresh
// date and refresh when it's within the configured window before the
// theoretical expiry (60 days from last refresh).
func (s *Service) maybeRefreshToken(accessToken, appSecret string, refreshDaysBefore int) error {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	// Conservative estimate: tokens expire 60 days after issuance.
	daysSinceRefresh := int(time.Since(s.tokenRefreshedAt).Hours() / 24)
	daysUntilEstimatedExpiry := 60 - daysSinceRefresh

	// Not close enough to expiry — skip.
	if daysUntilEstimatedExpiry > refreshDaysBefore {
		return nil
	}

	s.logger.Info("instagram token approaching estimated expiry, refreshing",
		zap.Int("days_since_refresh", daysSinceRefresh),
		zap.Int("days_until_expiry", daysUntilEstimatedExpiry))

	newToken, err := s.client.RefreshToken(context.Background(), appSecret)
	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	// Update the client's access token for all future API calls.
	s.tokenRefreshedAt = time.Now()
	s.client.accessToken = newToken

	s.logger.Info("instagram token refreshed successfully")
	return nil
}
