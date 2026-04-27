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

	"github.com/ainsleyclark/godaily/internal/news"
)

// HackerNews defines the type that implements news.Fetcher.
type HackerNews struct {
	url string
}

var (
	_         news.Fetcher = &HackerNews{}
	htmlTagRe              = regexp.MustCompile(`<[^>]*>`)
)

func init() {
	news.Register(news.SourceHN, NewHackerNews())
}

const hnURL = "https://hn.algolia.com/api/v1/search_by_date?query=golang&tags=story"

// NewHackerNews creates a Hacker News Algolia client.
func NewHackerNews() *HackerNews {
	return &HackerNews{
		url: hnURL,
	}
}

// Fetch retrieves all news items from Hacker News via the Algolia search API.
func (h HackerNews) Fetch(ctx context.Context) ([]news.Item, error) {
	response, err := fetch[hnResponse](ctx, h.url, "hacker news", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	return transformAll(response.Hits), nil
}

func (h hnHit) shouldInclude() bool { return true }

// transform maps an hnHit to a news.Item.
//
// If the story has no external URL (Ask HN / self-posts), it falls back to the
// HN permalink: https://news.ycombinator.com/item?id=<objectID>
func (h hnHit) transform() news.Item {
	u := h.URL
	if u == "" {
		u = "https://news.ycombinator.com/item?id=" + h.ObjectID
	}
	return news.Item{
		Source:    news.SourceHN,
		Title:     h.Title,
		URL:       u,
		Author:    h.Author,
		Snippet:   sanitiseSnippet(h.StoryText),
		Tag:       news.TagArticle,
		Comments:  h.NumComments,
		Published: h.CreatedAt,
	}
}

// sanitiseSnippet strips HTML tags and unescapes HTML entities from the
// story_text field, which the Algolia HN API returns as raw HTML.
func sanitiseSnippet(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

type (
	hnResponse struct {
		Hits []hnHit `json:"hits"`
	}
	hnHit struct {
		ObjectID    string    `json:"objectID"`
		Title       string    `json:"title"`
		URL         string    `json:"url"`
		Author      string    `json:"author"`
		StoryText   string    `json:"story_text"`
		Points      int       `json:"points"`
		NumComments int       `json:"num_comments"`
		CreatedAt   time.Time `json:"created_at"`
	}
)
