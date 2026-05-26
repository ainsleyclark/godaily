// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// Mastodon defines the type that implements news.Fetcher for the public
// mastodon.social #golang timeline.
type Mastodon struct {
	url string
}

var _ news.Fetcher = &Mastodon{}

func init() {
	news.Register(news.SourceMastodon, func(cfg env.Config) news.Fetcher { return NewMastodon(cfg) })
}

const (
	mastodonURL           = "https://mastodon.social/api/v1/timelines/tag/golang?limit=40"
	mastodonMinFavourites = 3
	mastodonTitleMaxLen   = 80
)

// NewMastodon creates a Mastodon hashtag-timeline client.
func NewMastodon(_ env.Config) *Mastodon {
	return &Mastodon{url: mastodonURL}
}

// Fetch retrieves recent #golang statuses from mastodon.social.
func (m Mastodon) Fetch(ctx context.Context) ([]news.Item, error) {
	statuses, err := ingest.Fetch[[]mastodonStatus](ctx, m.url, "mastodon", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, statuses), nil
}

// ShouldInclude filters out boosts (which duplicate other instances' posts),
// low-engagement noise, and non-English posts. The favourites threshold is a
// coarse but cheap proxy for "someone besides the author cared".
func (s mastodonStatus) ShouldInclude() bool {
	if s.Language != "" && s.Language != "en" {
		return false
	}
	return s.Reblog == nil && s.FavouritesCount >= mastodonMinFavourites
}

// EnrichmentURL is empty: Mastodon URLs point at the post itself, not at an
// external article with OG metadata worth crawling.
func (s mastodonStatus) EnrichmentURL() string { return "" }

// Transform maps a mastodonStatus to a news.Item. Mastodon posts have no
// title field, so the title is derived from the cleaned content.
func (s mastodonStatus) Transform() news.Item {
	var img string
	for _, m := range s.MediaAttachments {
		if m.Type == "image" && m.URL != "" {
			img = m.URL
			break
		}
	}
	return news.Item{
		Source: news.SourceMastodon,
		Title:  mastodonTitle(s.Content),
		URL:    s.URL,
		Author: &news.Author{
			Name:     s.Account.DisplayName,
			Username: s.Account.Username,
		},
		Snippet:   s.Content,
		ImageURL:  img,
		Tag:       news.TagDiscussion,
		Comments:  s.RepliesCount,
		Score:     news.ScoreOf(news.SourceMastodon, news.TagDiscussion, float64(s.FavouritesCount), true),
		Published: s.CreatedAt,
	}
}

var mastodonTagRe = regexp.MustCompile(`<[^>]*>`)

// mastodonTitle strips HTML, unescapes entities, takes the first line, and
// truncates to mastodonTitleMaxLen runes so titles stay one-liner-friendly.
func mastodonTitle(content string) string {
	clean := mastodonTagRe.ReplaceAllString(content, " ")
	clean = html.UnescapeString(clean)
	clean = strings.Join(strings.Fields(clean), " ")
	if i := strings.IndexAny(clean, "\n.!?"); i > 0 && i < mastodonTitleMaxLen {
		return strings.TrimSpace(clean[:i])
	}
	r := []rune(clean)
	if len(r) > mastodonTitleMaxLen {
		return strings.TrimSpace(string(r[:mastodonTitleMaxLen]))
	}
	return clean
}

type (
	mastodonStatus struct {
		ID               string               `json:"id"`
		CreatedAt        time.Time            `json:"created_at"`
		URL              string               `json:"url"`
		Content          string               `json:"content"`
		Language         string               `json:"language"`
		RepliesCount     int                  `json:"replies_count"`
		FavouritesCount  int                  `json:"favourites_count"`
		Reblog           *mastodonStatus      `json:"reblog"`
		Account          mastodonAccount      `json:"account"`
		MediaAttachments []mastodonAttachment `json:"media_attachments"`
	}
	mastodonAccount struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	mastodonAttachment struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}
)
