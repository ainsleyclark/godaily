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
	"net/http"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
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
func NewReddit() *Reddit {
	return &Reddit{url: redditURL}
}

// Fetch retrieves the latest posts from r/golang via the public JSON API.
func (r Reddit) Fetch(ctx context.Context) ([]news.Item, error) {
	listing, err := fetch[redditListing](ctx, r.url, "reddit", json.Unmarshal, http.Header{
		"User-Agent": {redditUserAgent},
	})
	if err != nil {
		return nil, err
	}
	return transformAll(listing.Data.Children), nil
}

// transform maps a redditChild to a news.Item.
//
// Self-posts have a URL pointing back to reddit.com/r/… rather than an
// external link. In that case we fall back to the full permalink.
func (c redditChild) shouldInclude() bool {
	return !strings.Contains(strings.ToLower(c.Data.Title), "help")
}

func (c redditChild) transform() news.Item {
	p := c.Data
	u := p.URL
	if strings.Contains(u, "reddit.com/r/") {
		u = "https://www.reddit.com" + p.Permalink
	}
	return news.Item{
		Source:    news.SourceReddit,
		Title:     p.Title,
		URL:       u,
		Author:    p.Author,
		Snippet:   strings.TrimSpace(p.SelfText),
		Score:     p.Score,
		Tag:       news.TagArticle,
		Comments:  p.NumComments,
		Published: time.Unix(int64(p.CreatedUTC), 0).UTC(),
	}
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
		Title       string  `json:"title"`
		URL         string  `json:"url"`
		Author      string  `json:"author"`
		SelfText    string  `json:"selftext"`
		Score       int     `json:"score"`
		NumComments int     `json:"num_comments"`
		CreatedUTC  float64 `json:"created_utc"`
		Permalink   string  `json:"permalink"`
	}
)
