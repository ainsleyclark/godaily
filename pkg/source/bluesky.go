// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ainsleydev/webkit/pkg/util/httputil"
	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
	"github.com/ainsleyclark/godaily/pkg/util/gohttp"
)

// Bluesky fetches recent #golang posts via the AT Protocol app.bsky.feed.searchPosts
// endpoint. Despite being documented as public, Bluesky now restricts searchPosts
// to authenticated callers (unauthenticated requests 403), so a session is created
// from BLUESKY_HANDLE + BLUESKY_APP_PASSWORD and the query is sent to the account's
// PDS, which proxies app.bsky.* reads to the AppView. No API key is involved — an
// app password is generated in Bluesky settings and is revocable.
type Bluesky struct {
	sessionURL  string
	searchURL   string
	handle      string
	appPassword string
	client      *http.Client
}

var _ news.Fetcher = &Bluesky{}

func init() {
	news.Register(news.SourceBluesky, func(cfg env.Config) news.Fetcher { return NewBluesky(cfg) })
}

const (
	// blueskyPDSBaseURL is the public PDS bsky-hosted accounts live on. It is
	// also where authenticated app.bsky.* reads are sent: the PDS proxies them
	// to the AppView server-side (no atproto-proxy header required).
	blueskyPDSBaseURL = "https://bsky.social"

	blueskySessionPath = "/xrpc/com.atproto.server.createSession"

	// sort=top surfaces the most-engaged posts first; lang=en narrows to
	// English at the source so the ingest language filter has less to drop.
	blueskySearchPath = "/xrpc/app.bsky.feed.searchPosts?q=%23golang&limit=40&sort=top&lang=en"

	blueskyUserAgent = "godaily/1.0 (+https://godaily.dev)"

	blueskyMinLikes = 3

	// blueskyPostCollection is the record type whose rkey forms the web URL.
	blueskyPostCollection = "app.bsky.feed.post"
)

// NewBluesky creates an authenticated Bluesky #golang search client.
func NewBluesky(cfg env.Config) *Bluesky {
	return &Bluesky{
		sessionURL:  blueskyPDSBaseURL + blueskySessionPath,
		searchURL:   blueskyPDSBaseURL + blueskySearchPath,
		handle:      cfg.BlueskyHandle,
		appPassword: cfg.BlueskyAppPassword,
		client:      gohttp.New(gohttp.WithTimeout(30 * time.Second)),
	}
}

// Fetch creates a session, then retrieves recent #golang posts. searchPosts
// requires authentication, so a missing handle/app password is a hard error
// rather than an empty result.
func (b Bluesky) Fetch(ctx context.Context) ([]news.Item, error) {
	if b.handle == "" || b.appPassword == "" {
		return nil, errors.New("bluesky: BLUESKY_HANDLE and BLUESKY_APP_PASSWORD must be set — searchPosts requires an authenticated session")
	}

	token, err := b.createSession(ctx)
	if err != nil {
		return nil, err
	}

	headers := http.Header{
		"Authorization": {"Bearer " + token},
		"User-Agent":    {blueskyUserAgent},
		"Accept":        {"application/json"},
	}
	response, err := ingest.Fetch[blueskySearchResponse](ctx, b.searchURL, "bluesky", json.Unmarshal, headers)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, response.Posts), nil
}

// createSession exchanges the handle and app password for a short-lived access
// JWT via com.atproto.server.createSession. A fresh session is created per
// fetch — the token expires within minutes, which is fine for a one-shot run.
func (b Bluesky) createSession(ctx context.Context) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"identifier": b.handle,
		"password":   b.appPassword,
	})
	if err != nil {
		return "", errors.Wrap(err, "bluesky session payload")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.sessionURL, bytes.NewReader(payload))
	if err != nil {
		return "", errors.Wrap(err, "bluesky session request creation failed")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", blueskyUserAgent)

	resp, err := b.client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "bluesky createSession")
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		return "", errors.Errorf("bluesky createSession: unexpected status code %d — check BLUESKY_HANDLE/BLUESKY_APP_PASSWORD", resp.StatusCode)
	}

	var out struct {
		AccessJWT string `json:"accessJwt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", errors.Wrap(err, "decoding bluesky session")
	}
	if out.AccessJWT == "" {
		return "", errors.New("bluesky createSession returned an empty accessJwt")
	}
	return out.AccessJWT, nil
}

// ShouldInclude drops empty posts, low-engagement noise, and non-English posts.
// The likes threshold is a cheap proxy for "someone besides the author cared".
func (p blueskyPost) ShouldInclude() bool {
	if strings.TrimSpace(p.Record.Text) == "" {
		return false
	}
	if !blueskyIsEnglish(p.Record.Langs) {
		return false
	}
	return p.LikeCount >= blueskyMinLikes
}

// EnrichmentURL is empty: a Bluesky URL points at the post itself, not at an
// external article with OG metadata worth crawling.
func (p blueskyPost) EnrichmentURL() string { return "" }

// Transform maps a blueskyPost to a news.Item. Posts have no title field, so
// the title is derived from the post text.
func (p blueskyPost) Transform() news.Item {
	var img string
	for _, image := range p.Embed.Images {
		if image.Fullsize != "" {
			img = image.Fullsize
			break
		}
		if image.Thumb != "" {
			img = image.Thumb
			break
		}
	}
	return news.Item{
		Source: news.SourceBluesky,
		Title:  mastodonTitle(p.Record.Text), // shared HTML-strip/first-line/truncate helper
		URL:    blueskyPostURL(p.Author.Handle, p.URI),
		Author: &news.Author{
			Name:       p.Author.DisplayName,
			Username:   p.Author.Handle,
			AvatarURL:  p.Author.Avatar,
			ProfileURL: blueskyProfileURL(p.Author.Handle),
		},
		Snippet:   p.Record.Text,
		ImageURL:  img,
		Tag:       news.TagSocial,
		Comments:  p.ReplyCount,
		Score:     news.ScoreOf(news.SourceBluesky, news.TagSocial, float64(p.LikeCount), true),
		Published: p.Record.CreatedAt,
	}
}

// blueskyIsEnglish reports whether the post's declared languages include
// English. Posts with no declared language are kept — the ingest language
// detector acts as a backstop.
func blueskyIsEnglish(langs []string) bool {
	if len(langs) == 0 {
		return true
	}
	for _, l := range langs {
		// langs may be region-qualified (e.g. "en-GB"); match the base tag.
		if l == "en" || strings.HasPrefix(l, "en-") {
			return true
		}
	}
	return false
}

// blueskyProfileURL builds the public profile URL for a handle.
func blueskyProfileURL(handle string) string {
	if handle == "" {
		return ""
	}
	return "https://bsky.app/profile/" + handle
}

// blueskyPostURL converts an AT URI (at://<did>/app.bsky.feed.post/<rkey>) and
// the author's handle into a browsable bsky.app post URL. Returns "" when the
// URI is not a feed post or the handle is missing.
func blueskyPostURL(handle, uri string) string {
	if handle == "" {
		return ""
	}
	i := strings.Index(uri, blueskyPostCollection+"/")
	if i < 0 {
		return ""
	}
	rkey := uri[i+len(blueskyPostCollection)+1:]
	if rkey == "" {
		return ""
	}
	return "https://bsky.app/profile/" + handle + "/post/" + rkey
}

type (
	blueskySearchResponse struct {
		Posts []blueskyPost `json:"posts"`
	}
	blueskyPost struct {
		URI        string        `json:"uri"`
		Author     blueskyAuthor `json:"author"`
		Record     blueskyRecord `json:"record"`
		Embed      blueskyEmbed  `json:"embed"`
		ReplyCount int           `json:"replyCount"`
		LikeCount  int           `json:"likeCount"`
	}
	blueskyAuthor struct {
		Handle      string `json:"handle"`
		DisplayName string `json:"displayName"`
		Avatar      string `json:"avatar"`
	}
	blueskyRecord struct {
		Text      string    `json:"text"`
		CreatedAt time.Time `json:"createdAt"`
		Langs     []string  `json:"langs"`
	}
	blueskyEmbed struct {
		Images []blueskyImage `json:"images"`
	}
	blueskyImage struct {
		Thumb    string `json:"thumb"`
		Fullsize string `json:"fullsize"`
	}
)
