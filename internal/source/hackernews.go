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
	"time"

	"github.com/ainsleyclark/godaily/internal/ingest"
	"github.com/ainsleyclark/godaily/internal/news"
)

// HackerNews defines the type that implements news.Fetcher.
type HackerNews struct {
	url string
}

var _ news.Fetcher = &HackerNews{}

func init() {
	news.Register(news.SourceHN, NewHackerNews())
}

const hnBaseURL = "https://hn.algolia.com/api/v1/search_by_date?query=golang&tags=story&hitsPerPage=50"

// NewHackerNews creates a Hacker News Algolia client.
func NewHackerNews() *HackerNews {
	return &HackerNews{
		url: hnYesterdayURL(),
	}
}

// hnYesterdayURL builds an Algolia URL that restricts results to yesterday UTC,
// so collect always retrieves the correct day's items regardless of run time.
// Uses strict inequalities: >start-1 and <end+1 to cover the full 24-hour window.
func hnYesterdayURL() string {
	day := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	next := day.Add(24 * time.Hour)
	return fmt.Sprintf("%s&numericFilters=created_at_i>%d,created_at_i<%d", hnBaseURL, day.Unix()-1, next.Unix())
}

// Fetch retrieves all news items from Hacker News via the Algolia search API.
func (h HackerNews) Fetch(ctx context.Context) ([]news.Item, error) {
	response, err := ingest.Fetch[hnResponse](ctx, h.url, "hacker news", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, response.Hits), nil
}

func (h hnHit) ShouldInclude() bool { return true }

// EnrichmentURL returns the external story URL for crawling, or "" for
// Ask-HN / self-posts (which fall back to news.ycombinator.com/item?id=).
func (h hnHit) EnrichmentURL() string { return h.URL }

// Transform maps an hnHit to a news.Item.
//
// URL is the external story URL (the click target). OriginalURL is the HN
// permalink where the story was posted. For Ask HN / self-posts the story has
// no external URL, so URL falls back to the HN permalink and OriginalURL stays
// empty (it would duplicate URL).
func (h hnHit) Transform() news.Item {
	hnPermalink := "https://news.ycombinator.com/item?id=" + h.ObjectID
	u := h.URL
	original := hnPermalink
	if u == "" {
		u = hnPermalink
		original = ""
	}
	return news.Item{
		Source:      news.SourceHN,
		Title:       h.Title,
		URL:         u,
		OriginalURL: original,
		Author: &news.Author{
			Username:   h.Author,
			ProfileURL: "https://news.ycombinator.com/user?id=" + h.Author,
		},
		Snippet:   h.StoryText,
		Tag:       news.TagArticle,
		Comments:  h.NumComments,
		Score:     news.ScoreOf(news.SourceHN, news.TagArticle, float64(h.Points), true),
		Published: h.CreatedAt,
	}
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
