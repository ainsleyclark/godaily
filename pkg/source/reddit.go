// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// Reddit defines the type that implements news.Fetcher for r/golang.
//
// By default posts come from Arctic Shift (see NewReddit), a free, keyless
// Reddit archive that indexes r/golang within seconds of posting and returns
// each post in Reddit's own JSON shape. The previous ScraperAPI → reddit.com
// path is retained behind NewRedditScraperAPI but is no longer registered.
type Reddit struct {
	url string
	// viaScraperAPI selects the legacy path: url is Reddit's own listing
	// (proxied through ScraperAPI) rather than the Arctic Shift search API.
	viaScraperAPI bool
	// now returns the current time; it anchors the Arctic Shift look-back
	// window. Nil means time.Now.
	now func() time.Time
}

var _ news.Fetcher = &Reddit{}

func init() {
	news.Register(news.SourceReddit, func(cfg env.Config) news.Fetcher { return NewReddit(cfg) })
}

const (
	// arcticShiftURL is the Arctic Shift post search for r/golang, newest
	// first. limit=100 is the API maximum; Fetch adds an "after" bound so only
	// the last arcticShiftLookback of posts are requested.
	// Docs: https://github.com/ArthurHeitmann/arctic_shift/tree/master/api
	arcticShiftURL = "https://arctic-shift.photon-reddit.com/api/posts/search?subreddit=golang&sort=desc&limit=100"
	// arcticShiftLookback bounds the request to roughly what Reddit's own
	// /new?limit=25 listing used to cover (~2 days of r/golang). Collection
	// keeps only yesterday's items, so anything older would just be crawled
	// for enrichment and then discarded.
	arcticShiftLookback = 48 * time.Hour
	// arcticShiftUserAgent identifies GoDaily to the Arctic Shift maintainers.
	arcticShiftUserAgent = "godaily/1.0 (+https://godaily.dev)"

	// redditURL is Reddit's own r/golang listing, used only by the legacy
	// ScraperAPI path (NewRedditScraperAPI).
	redditURL = "https://www.reddit.com/r/golang/new.json?limit=25"
	// redditUserAgent mimics a real Chrome on Windows so it stays consistent
	// with the Sec-CH-UA client hints below; Reddit 403s requests whose
	// User-Agent and client hints don't look like a genuine browser.
	redditUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36"
	// redditCookie is a logged-out (anonymous) device identifier Reddit issues
	// to browsers. It carries no account credentials. If Reddit tightens
	// protection further, refresh it from a real browser request to reddit.com.
	redditCookie = "loid=0000000000gniio6du.2.1637187642553.Z0FBQUFBQm54d2FCZDA3MFdzN2NN" +
		"UDI1ZjdDV1hqbWlheTdBQlhvVlVEMFdTUERzWFlXeC1oZUtpaDVBRUVYM2pIVHFJZWdDcm8tMjZWeUhCdG9LN3" +
		"FldVZUSkVCaVZrdHJ0VzZqWU90YllUdHBfb2g5WWdXUDJ4Q0w4QUtFWXdHdmdHdFJLb3dFVHI"
)

// NewReddit creates a Reddit client that reads r/golang from Arctic Shift.
//
// Arctic Shift replaced the ScraperAPI route in November 2026: Reddit began
// blocking ScraperAPI's premium (residential) pool — every request hung ~55s
// and returned a 500 — and the ultra_premium pool that ScraperAPI suggests
// needs a paid plan. Arctic Shift is not Reddit, so datacenter IPs (Vercel)
// are not blocked and no key is needed. It is a community project without an
// uptime guarantee; bin/ingest-reddit.sh remains the manual fallback.
func NewReddit(_ env.Config) *Reddit {
	return &Reddit{url: arcticShiftURL}
}

// NewRedditScraperAPI creates a Reddit client that fetches reddit.com's own
// /new.json listing, routed through ScraperAPI when cfg.ScraperAPIKeys is set
// (to avoid IP blocks on hosts like Vercel and GitHub Actions). WithKeepHeaders
// ensures the browser-like headers below are forwarded to Reddit rather than
// stripped by the proxy.
//
// Deprecated: not registered as the Reddit source since Reddit started
// blocking ScraperAPI's premium pool (see NewReddit). It is kept, working and
// tested, so it can be switched back in news.Register if the proxy route
// recovers (e.g. with ultra_premium on a paid ScraperAPI plan).
func NewRedditScraperAPI(cfg env.Config) *Reddit {
	return &Reddit{
		url:           ingest.ScraperURL(cfg.ScraperAPIKeys, redditURL, ingest.WithKeepHeaders()),
		viaScraperAPI: true,
	}
}

// redditHeaders returns the browser-like request headers Reddit's .json
// endpoint expects (legacy ScraperAPI path only). Reddit 403s requests that
// don't carry a realistic browser fingerprint, so we replicate Chrome's
// headers (and an anonymous cookie).
//
// Accept-Encoding is deliberately omitted: Go's transport adds gzip and
// transparently decompresses the response, whereas setting it manually would
// force us to decode the body ourselves before json.Unmarshal.
func redditHeaders() http.Header {
	return http.Header{
		"User-Agent":                {redditUserAgent},
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"Accept-Language":           {"en-US,en;q=0.9"},
		"Cache-Control":             {"no-cache"},
		"Pragma":                    {"no-cache"},
		"Priority":                  {"u=0, i"},
		"Sec-Ch-Ua":                 {`"Chromium";v="148", "Google Chrome";v="148", "Not/A)Brand";v="99"`},
		"Sec-Ch-Ua-Mobile":          {"?0"},
		"Sec-Ch-Ua-Platform":        {`"Windows"`},
		"Sec-Fetch-Dest":            {"document"},
		"Sec-Fetch-Mode":            {"navigate"},
		"Sec-Fetch-Site":            {"cross-site"},
		"Sec-Fetch-User":            {"?1"},
		"Upgrade-Insecure-Requests": {"1"},
		"Cookie":                    {redditCookie},
	}
}

// Fetch retrieves the latest r/golang posts from Arctic Shift, or from
// Reddit's own listing when built with NewRedditScraperAPI.
func (r Reddit) Fetch(ctx context.Context) ([]news.Item, error) {
	if r.viaScraperAPI {
		return r.fetchListing(ctx)
	}
	return r.fetchArcticShift(ctx)
}

// fetchArcticShift queries the Arctic Shift search API. Its posts carry the
// same fields as Reddit's listing children, so they are wrapped as
// redditChild and run through the identical transform pipeline.
func (r Reddit) fetchArcticShift(ctx context.Context) ([]news.Item, error) {
	u, err := url.Parse(r.url)
	if err != nil {
		return nil, errors.Wrap(err, "parsing arctic shift url")
	}
	now := time.Now
	if r.now != nil {
		now = r.now
	}
	q := u.Query()
	q.Set("after", strconv.FormatInt(now().Add(-arcticShiftLookback).Unix(), 10))
	u.RawQuery = q.Encode()

	resp, err := ingest.Fetch[arcticShiftResponse](ctx, u.String(), "reddit", json.Unmarshal, http.Header{
		"User-Agent": {arcticShiftUserAgent},
		"Accept":     {"application/json"},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New("arctic shift: " + resp.Error)
	}

	children := make([]redditChild, len(resp.Data))
	for i, p := range resp.Data {
		children[i] = redditChild{Data: p}
	}
	return ingest.TransformAll(ctx, children), nil
}

// fetchListing retrieves reddit.com's own /new.json listing (legacy path).
func (r Reddit) fetchListing(ctx context.Context) ([]news.Item, error) {
	listing, err := ingest.Fetch[redditListing](ctx, r.url, "reddit", json.Unmarshal, redditHeaders())
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, listing.Data.Children), nil
}

// ParseReddit decodes a raw r/golang listing payload (the exact JSON body
// returned by https://www.reddit.com/r/golang/new.json) into transformed news
// items. It runs the same transform pipeline as Fetch, so manually submitted
// JSON is processed identically to a live fetch. Used by the submit endpoint
// as a fallback when ScraperAPI is blocked.
func ParseReddit(ctx context.Context, data []byte) ([]news.Item, error) {
	var listing redditListing
	if err := json.Unmarshal(data, &listing); err != nil {
		return nil, errors.Wrap(err, "decoding reddit listing")
	}
	return ingest.TransformAll(ctx, listing.Data.Children), nil
}

// ShouldInclude reports whether the post should appear in the digest.
// Removed or deleted posts are dropped (see isRemoved), as are posts whose
// title or body contains "help" or "feedback".
func (c redditChild) ShouldInclude() bool {
	if c.Data.isRemoved() {
		return false
	}
	title := strings.ToLower(c.Data.Title)
	body := strings.ToLower(c.Data.SelfText)
	return !strings.Contains(title, "help") &&
		!strings.Contains(title, "feedback") &&
		!strings.Contains(title, "learning") &&
		!strings.Contains(body, "feedback")
}

// redditRemovedPrefix is how Reddit rewrites the title of a removed post,
// e.g. "[ Removed by moderator ]" or "[ Removed by Reddit ]".
const redditRemovedPrefix = "[ Removed by"

// redditAutomodFiltered is the removal category for posts AutoModerator holds
// for review. Mods routinely approve these, so it is not treated as removal.
const redditAutomodFiltered = "automod_filtered"

// isRemoved reports whether the post has been removed by moderators or Reddit,
// or deleted by its author.
//
// Reddit's own listing hides such posts, but Arctic Shift archives every post
// as soon as it is submitted and records removals afterwards, so they must be
// filtered here. The rules were checked against a real reddit.com /new
// capture and Arctic Shift's records for the same 54-hour window (see
// testdata/arcticshift): they drop 10 of the 17 posts Reddit hid and none of
// the 25 it showed. automod_filtered and a "[removed]" body are deliberately
// ignored: in that capture Reddit showed six posts with those markers (held
// for review, then approved). The remaining hidden posts carry no removal
// marker at all, so no field-based rule can match them.
//
// Posts from Reddit's own listing never carry these markers, so the filter is
// a no-op on the legacy path and on manually ingested listings.
func (p redditPost) isRemoved() bool {
	if p.RemovedByCategory != "" && p.RemovedByCategory != redditAutomodFiltered {
		return true
	}
	if p.Meta.RemovalType != "" && p.Meta.RemovalType != redditAutomodFiltered {
		return true
	}
	if p.Meta.WasDeletedLater {
		return true
	}
	if strings.HasPrefix(p.Title, redditRemovedPrefix) || strings.HasPrefix(p.Meta.EditedTitle, redditRemovedPrefix) {
		return true
	}
	return p.Author == "[deleted]" || p.SelfText == "[deleted]"
}

// EnrichmentURL returns the external URL for crawler enrichment, or "" for
// self-posts (which point back to reddit.com and have no useful meta tags).
func (c redditChild) EnrichmentURL() string {
	if c.Data.URL == "" || strings.Contains(c.Data.URL, "reddit.com/r/") {
		return ""
	}
	return c.Data.URL
}

// Transform maps a redditChild to a news.Item.
//
// For link posts the external URL is the click target (URL) and the Reddit
// thread is stored as OriginalURL so "Read on Reddit" navigates to Reddit.
// Self-posts point directly at the Reddit thread with no OriginalURL.
func (c redditChild) Transform() news.Item {
	p := c.Data
	permalink := "https://www.reddit.com" + p.Permalink
	u := p.URL
	var originalURL string
	if strings.Contains(u, "reddit.com/r/") {
		u = permalink
	} else {
		originalURL = permalink
	}
	return news.Item{
		Source:      news.SourceReddit,
		Title:       p.Title,
		URL:         u,
		OriginalURL: originalURL,
		ImageURL:    redditImage(p),
		Author: &news.Author{
			Username:   p.Author,
			ProfileURL: "https://www.reddit.com/user/" + p.Author,
		},
		Snippet:   redditSnippet(p.SelfText),
		Tag:       news.TagDiscussion,
		Comments:  p.NumComments,
		Score:     news.ScoreOf(news.SourceReddit, news.TagDiscussion, float64(p.Score), true),
		Published: time.Unix(int64(p.CreatedUTC), 0).UTC(),
	}
}

// redditRemovedBody is the placeholder Reddit substitutes for the body of a
// removed or filtered post.
const redditRemovedBody = "[removed]"

// redditSnippet returns the post body for use as a snippet, or "" when the body
// is Reddit's "[removed]" placeholder. Arctic Shift archives a post the moment
// it is submitted, so a post AutoModerator held for review (and mods later
// approved) can carry "[removed]" instead of its real text; enrichment then
// fills the snippet from the linked page instead.
func redditSnippet(selfText string) string {
	if strings.TrimSpace(selfText) == redditRemovedBody {
		return ""
	}
	return selfText
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
		ID          string        `json:"id"`
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
		// RemovedByCategory is null on visible posts and set (e.g.
		// "moderator", "reddit", "deleted", "automod_filtered") on removed
		// ones. Reddit's listing only exposes visible posts; Arctic Shift
		// records removals.
		RemovedByCategory string `json:"removed_by_category"`
		// Meta is Arctic Shift's bookkeeping about later re-fetches of the
		// post; absent from Reddit's own listing.
		Meta arcticShiftMeta `json:"_meta"`
	}
	// arcticShiftMeta is the subset of Arctic Shift's per-post "_meta" object
	// that records what happened to the post after it was first archived.
	arcticShiftMeta struct {
		RemovalType     string `json:"removal_type"`
		EditedTitle     string `json:"edited_title"`
		WasDeletedLater bool   `json:"was_deleted_later"`
	}
	// arcticShiftResponse is the envelope returned by Arctic Shift's
	// /api/posts/search. Posts sit directly in data (not data.children[].data
	// as on reddit.com); on failure data is null and error is set.
	arcticShiftResponse struct {
		Data  []redditPost `json:"data"`
		Error string       `json:"error"`
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
