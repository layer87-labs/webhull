// Package instagram provides Instagram feed functionality for webhull.
// It fetches posts via the Instagram Graph API, caches them in memory,
// and exposes them for server-side rendering in page templates.
package instagram

import "time"

// SelectionMode determines which posts are shown in the feed.
type SelectionMode string

const (
	// ModeLatestN shows the most recent N posts by timestamp.
	ModeLatestN SelectionMode = "latest_n"

	// ModeTopEngagement shows top N posts by like_count + comments_count
	// from the latest fetched batch.
	ModeTopEngagement SelectionMode = "top_engagement"

	// ModeManual shows only posts listed in ManualPostIDs (static curation).
	// No API calls are made to discover posts, but engagement metrics
	// (current like/comment counts) are still fetched via API if desired.
	ModeManual SelectionMode = "manual"
)

// Post represents a single Instagram media post for template rendering.
type Post struct {
	// ID is the Instagram media ID (e.g. "18012345678901234").
	ID string

	// Caption is the post caption text. May be empty.
	Caption string

	// MediaType is the Instagram media type: IMAGE, VIDEO, CAROUSEL_ALBUM.
	MediaType string

	// MediaURL is the CDN URL to the media file (image or video still frame).
	// These URLs expire after several hours.
	MediaURL string

	// ThumbnailURL is the thumbnail URL (set for VIDEO posts only).
	ThumbnailURL string

	// Permalink is the public instagram.com URL for this post.
	Permalink string

	// Timestamp is when the post was created (ISO 8601 / RFC 3339).
	Timestamp time.Time

	// Username is the Instagram username of the post owner.
	Username string

	// LikeCount is the number of likes. May be 0 if hidden.
	LikeCount int

	// CommentsCount is the number of comments. May be 0 if disabled.
	CommentsCount int

	// MediaProductType is the surface where the media was published: FEED, STORY, REELS, IGTV.
	MediaProductType string

	// Children holds child media IDs for CAROUSEL_ALBUM posts.
	// The first child's media_url is used as the display image.
	Children []ChildMedia
}

// ChildMedia represents a single child media item within a CAROUSEL_ALBUM post.
type ChildMedia struct {
	// ID is the child media ID.
	ID string

	// MediaURL is the CDN URL of the child media file.
	MediaURL string

	// MediaType is the media type of the child: IMAGE or VIDEO.
	MediaType string
}

// FeedRequest contains the parameters to request a feed.
type FeedRequest struct {
	// SelectionMode controls which posts are returned.
	SelectionMode SelectionMode

	// Count is the number of posts to return (1-25).
	Count int

	// FetchMultiplier controls how many posts to fetch from the API before
	// local filtering/ranking. API limit from IG is the default page size.
	FetchMultiplier int

	// ManualPostIDs is used when SelectionMode is ModeManual.
	ManualPostIDs []string

	// ExcludeVideo filters out VIDEO and IGTV posts when true.
	ExcludeVideo bool

	// FilterMediaProductType limits posts to these surfaces (e.g. ["FEED"]).
	FilterMediaProductType []string
}

// cachedFeed holds the last fetch result with its timestamp.
type cachedFeed struct {
	Posts     []Post
	FetchedAt time.Time
}
