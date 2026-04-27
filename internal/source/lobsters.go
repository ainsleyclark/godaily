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
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
)

// Lobsters defines the type that implements news.Fetcher for lobste.rs/t/go.
type Lobsters struct {
	url string
}

var _ news.Fetcher = &Lobsters{}

func init() {
	news.Register(news.SourceLobsters, func() news.Fetcher { return NewLobsters() })
}

const lobstersURL = "https://lobste.rs/t/go.json"

// NewLobsters creates a Lobsters client targeting the Go tag.
func NewLobsters() *Lobsters {
	return &Lobsters{url: lobstersURL}
}

// Fetch retrieves the latest Go-tagged stories from lobste.rs.
func (l Lobsters) Fetch(ctx context.Context) ([]news.Item, error) {
	stories, err := fetch[[]lobstersStory](ctx, l.url, "lobsters", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	return transformAll(stories), nil
}

func (s lobstersStory) shouldInclude() bool { return true }

func (s lobstersStory) transform() news.Item {
	published, _ := time.Parse(time.RFC3339, s.CreatedAt)
	return news.Item{
		Source:    news.SourceLobsters,
		Title:     s.Title,
		URL:       s.URL,
		Author:    s.SubmitterUser,
		Snippet:   strings.TrimSpace(s.Description),
		Score:     s.Score,
		Tag:       news.TagArticle,
		Comments:  s.CommentCount,
		Published: published.UTC(),
	}
}

type lobstersStory struct {
	ShortID       string   `json:"short_id"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Score         int      `json:"score"`
	CommentCount  int      `json:"comment_count"`
	CreatedAt     string   `json:"created_at"`
	Description   string   `json:"description"`
	SubmitterUser string   `json:"submitter_user"`
	Tags          []string `json:"tags"`
}
