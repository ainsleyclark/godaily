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
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/ingest"
	"github.com/ainsleyclark/godaily/pkg/news"
)

// Mastodon defines the type that implements news.Fetcher for the public
// mastodon.social #golang timeline.
type Mastodon struct {
	url string
}

var _ news.Fetcher = &Mastodon{}

func init() {
	news.Register(news.SourceMastodon, NewMastodon())
}

const (
	mastodonURL           = "https://mastodon.social/api/v1/timelines/tag/golang?limit=40"
	mastodonMinFavourites = 3
	mastodonTitleMaxLen   = 80
)

// NewMastodon creates a Mastodon hashtag-timeline client.
func NewMastodon() *Mastodon {
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

// ShouldInclude filters out boosts (which duplicate other instances' posts)
// and low-engagement noise. The favourites threshold is a coarse but cheap
// proxy for "someone besides the author cared".
func (s mastodonStatus) ShouldInclude() bool {
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
