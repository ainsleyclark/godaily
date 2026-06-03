// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// Bluesky fetches recent #golang posts from the public Bluesky AppView using
// the app.bsky.feed.searchPosts endpoint. public.api.bsky.app serves
// unauthenticated read traffic, so no session token or app password is needed.
type Bluesky struct {
	url string
}

var _ news.Fetcher = &Bluesky{}

func init() {
	news.Register(news.SourceBluesky, func(cfg env.Config) news.Fetcher { return NewBluesky(cfg) })
}

const (
	// sort=top surfaces the most-engaged posts first; lang=en narrows to
	// English at the source so the ingest language filter has less to drop.
	blueskyURL = "https://public.api.bsky.app/xrpc/app.bsky.feed.searchPosts" +
		"?q=%23golang&limit=40&sort=top&lang=en"

	blueskyMinLikes = 3

	// blueskyPostCollection is the record type whose rkey forms the web URL.
	blueskyPostCollection = "app.bsky.feed.post"
)

// NewBluesky creates a Bluesky #golang search client.
func NewBluesky(_ env.Config) *Bluesky {
	return &Bluesky{url: blueskyURL}
}

// Fetch retrieves recent #golang posts from the public Bluesky AppView.
func (b Bluesky) Fetch(ctx context.Context) ([]news.Item, error) {
	response, err := ingest.Fetch[blueskySearchResponse](ctx, b.url, "bluesky", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, response.Posts), nil
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
