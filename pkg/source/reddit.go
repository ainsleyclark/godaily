// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package source

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/ingest"
	"github.com/ainsleyclark/godaily/pkg/news"
)

// Reddit defines the type that implements news.Fetcher for r/golang.
type Reddit struct {
	url string
}

var _ news.Fetcher = &Reddit{}

func init() {
	news.Register(news.SourceReddit, NewReddit())
}

const (
	redditURL       = "https://www.reddit.com/r/golang/new.json?limit=25"
	redditUserAgent = "godaily/1.0"
)

// NewReddit creates a Reddit client targeting r/golang.
// If SCRAPER_API_KEY is set, requests are routed through ScraperAPI to avoid
// IP blocks on restricted hosting environments (e.g. Vercel, GitHub Actions).
func NewReddit() *Reddit {
	u := redditURL
	if key := os.Getenv("SCRAPER_API_KEY"); key != "" {
		u = fmt.Sprintf("http://api.scraperapi.com?api_key=%s&url=%s", key, url.QueryEscape(redditURL))
		slog.Info("reddit: routing fetch through ScraperAPI")
	}
	return &Reddit{url: u}
}

// Fetch retrieves the latest posts from r/golang via the public JSON API.
func (r Reddit) Fetch(ctx context.Context) ([]news.Item, error) {
	listing, err := ingest.Fetch[redditListing](ctx, r.url, "reddit", json.Unmarshal, http.Header{
		"User-Agent": {redditUserAgent},
	})
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, listing.Data.Children), nil
}

// ShouldInclude reports whether the post should appear in the digest.
// Posts whose title or body contains "help" or "feedback" are dropped.
func (c redditChild) ShouldInclude() bool {
	title := strings.ToLower(c.Data.Title)
	body := strings.ToLower(c.Data.SelfText)
	return !strings.Contains(title, "help") &&
		!strings.Contains(title, "feedback") &&
		!strings.Contains(body, "feedback")
}

// Transform maps a redditChild to a news.Item.
//
// Self-posts have a URL pointing back to reddit.com/r/… rather than an
// external link. In that case we fall back to the full permalink.

// EnrichmentURL returns the external URL for crawler enrichment, or "" for
// self-posts (which point back to reddit.com and have no useful meta tags).
func (c redditChild) EnrichmentURL() string {
	if c.Data.URL == "" || strings.Contains(c.Data.URL, "reddit.com/r/") {
		return ""
	}
	return c.Data.URL
}

func (c redditChild) Transform() news.Item {
	p := c.Data
	u := p.URL
	if strings.Contains(u, "reddit.com/r/") {
		u = "https://www.reddit.com" + p.Permalink
	}
	return news.Item{
		Source:   news.SourceReddit,
		Title:    p.Title,
		URL:      u,
		ImageURL: redditImage(p),
		Author: &news.Author{
			Username:   p.Author,
			ProfileURL: "https://www.reddit.com/user/" + p.Author,
		},
		Snippet:   p.SelfText,
		Tag:       news.TagDiscussion,
		Comments:  p.NumComments,
		Score:     news.ScoreOf(news.SourceReddit, news.TagDiscussion, float64(p.Score), true),
		Published: time.Unix(int64(p.CreatedUTC), 0).UTC(),
	}
}

// redditThumbnailSentinels are the placeholder values Reddit returns in the
// thumbnail field when there's no usable image.
var redditThumbnailSentinels = map[string]bool{
	"self": true, "default": true, "nsfw": true, "spoiler": true, "image": true, "": true,
}

// redditImage extracts the best available image URL from a redditPost,
// preferring the high-resolution preview source over the low-res thumbnail.
// Reddit returns HTML-escaped URLs (e.g. &amp;) so we unescape them.
func redditImage(p redditPost) string {
	if len(p.Preview.Images) > 0 {
		if u := p.Preview.Images[0].Source.URL; u != "" {
			return html.UnescapeString(u)
		}
	}
	if redditThumbnailSentinels[p.Thumbnail] {
		return ""
	}
	if strings.HasPrefix(p.Thumbnail, "http://") || strings.HasPrefix(p.Thumbnail, "https://") {
		return p.Thumbnail
	}
	return ""
}

type (
	redditListing struct {
		Data redditListingData `json:"data"`
	}
	redditListingData struct {
		Children []redditChild `json:"children"`
	}
	redditChild struct {
		Data redditPost `json:"data"`
	}
	redditPost struct {
		Title       string        `json:"title"`
		URL         string        `json:"url"`
		Author      string        `json:"author"`
		SelfText    string        `json:"selftext"`
		Score       int           `json:"score"`
		NumComments int           `json:"num_comments"`
		CreatedUTC  float64       `json:"created_utc"`
		Permalink   string        `json:"permalink"`
		Preview     redditPreview `json:"preview"`
		Thumbnail   string        `json:"thumbnail"`
	}
	redditPreview struct {
		Images []redditPreviewImage `json:"images"`
	}
	redditPreviewImage struct {
		Source redditPreviewSource `json:"source"`
	}
	redditPreviewSource struct {
		URL string `json:"url"`
	}
)
