# Instagram Feed

Webhull can fetch and display recent Instagram posts directly on your website.
The feed is **server-side rendered** — posts are present in the initial HTML
for SEO and crawlers — with lazy-loaded images for performance.

## How it works

1. A background sync job fetches posts from the [Instagram Graph API](https://developers.facebook.com/docs/instagram-platform/instagram-api-with-facebook-login) at a configurable interval (default: every 15 minutes).
2. Posts are cached in memory and served instantly on page requests.
3. On API failure, the last valid cache state is served — your site never breaks because Instagram is down.
4. Long-lived access tokens are **automatically refreshed** in the background before expiry.

## Setup

### 1. Create a Meta App

1. Go to [Meta for Developers](https://developers.facebook.com/) → **Create App**
2. Choose app type: **Business**
3. Add the **Instagram Graph API** product to your app
4. Under **App Review → Permissions and Features**, request:
   - `instagram_basic` — basic access to profile
   - `pages_read_engagement` — access to media and engagement data

### 2. Connect an Instagram Account

1. In your Meta app, go to **Instagram → Instagram API with Facebook Login** → **Generate Token**
2. Log in with the Facebook account that manages the Instagram Business/Creator account
3. Select the connected Facebook Page and Instagram account
4. You'll receive a **short-lived access token** (valid 1 hour)

### 3. Exchange for a Long-Lived Token

Long-lived tokens are valid for ~60 days and can be refreshed programmatically. Exchange the short-lived token:

```bash
curl "https://graph.instagram.com/access_token?grant_type=ig_exchange_token&client_secret=YOUR_APP_SECRET&access_token=SHORT_LIVED_TOKEN"
```

The response contains a long-lived `access_token`.

### 4. Find Your Instagram User ID

```bash
curl "https://graph.instagram.com/v25.0/me?fields=id,username&access_token=LONG_LIVED_TOKEN"
```

The response `id` is your **Instagram user ID** (numeric string).

### 5. Configure Webhull

Add to `pages.yaml` (site structure, baked into container image):

```yaml
instagram:
  enabled: true
  accessToken: "${INSTAGRAM_ACCESS_TOKEN}"
  userID: "${INSTAGRAM_USER_ID}"
  appSecret: "${INSTAGRAM_APP_SECRET}"
  selectionMode: "latest_n"   # latest_n | top_engagement | manual
  count: 6                     # posts to display
  cacheTTL: "15m"              # how often to poll the API
  excludeVideo: true           # show only images (video posts get a thumbnail + link)
```

Provide secrets at runtime via environment variables in your `config.yaml` or Helm values:

```yaml
env:
  INSTAGRAM_ACCESS_TOKEN: "your-long-lived-token"
  INSTAGRAM_USER_ID: "123456789"
  INSTAGRAM_APP_SECRET: "abc123..."
```

> **Never** put the access token, user ID, or app secret in source-controlled config files.
> Always use `${VAR}` expansion.

## Selection Modes

| Mode | Description | API calls |
|------|-------------|-----------|
| `latest_n` | Most recent N posts by publish date | 1 per cache interval |
| `top_engagement` | Top N by likes + comments from the latest batch | 1 per cache interval |
| `manual` | Static list of specific post IDs | N (one per post ID) |

Example with manual mode:

```yaml
instagram:
  selectionMode: "manual"
  manualPostIDs:
    - "18012345678901234"
    - "18098765432109876"
  count: 2
```

## Template

The Instagram feed renders as a responsive grid (3 columns on desktop, 2 on tablet, 1 on mobile).
It is automatically included in the **home page** template when the feed has posts.

Each post card shows:
- Lazy-loaded image
- Caption excerpt (truncated to 3 lines)
- Relative timestamp ("2 days ago")
- Heart + comment counts (when > 0)
- Link to the original Instagram post
- Video overlay icon for VIDEO posts
- Carousel icon for CAROUSEL_ALBUM posts

All images use `loading="lazy"` for performance. The HTML structure is present on first render
— no JavaScript required for content visibility.

## Token Lifecycle

- Long-lived tokens expire after **~60 days**
- Webhull refreshes the token in the background **7 days before expiry**
- If the refresh fails, it retries on the next daily check
- Your site continues serving the last cached posts while the token issue is resolved

## Rate Limits

The Instagram Graph API allows approximately 240 calls per hour.
With a default `cacheTTL` of 15 minutes (4 calls/hour), you are well within limits.
Even with `top_engagement` mode fetching a larger batch, the multiplier only affects
the result set size, not the number of API calls.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Feed section doesn't appear | `instagram.enabled: true` not set | Check config |
| Feed empty after config change | Cold cache — first fetch is synchronous | Wait for the first page request |
| "instagram.accessToken is required" | Missing `${INSTAGRAM_ACCESS_TOKEN}` | Set the env var |
| "instagram.userID is required" | Missing user ID | Find ID with `/v25.0/me?fields=id` |
| API errors in logs | Token expired or invalid | Refresh token manually, update env var |
| Posts are stale | `cacheTTL` is too long | Reduce to `5m` |
| Video posts show still frame | Expected — videos need separate permission | Set `excludeVideo: true` |

## Architecture

```
pages.yaml (instagram config)
  ↓
config.InstagramConfig
  ↓
instagram.NewService()
  ├── Client  → Instagram Graph API (GET /{user-id}/media)
  ├── Cache   → []Post + fetchedAt (sync.RWMutex)
  └── Token   → background goroutine → daily refresh check
  ↓
buildPageData() → PageData.InstagramFeed
  ↓
instagram_feed.templ → responsive HTML grid
```