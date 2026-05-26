// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// Lobsters defines the type that implements news.Fetcher for lobste.rs/t/go.
type Lobsters struct {
	url string
}

var _ news.Fetcher = &Lobsters{}

func init() {
	news.Register(news.SourceLobsters, func(cfg env.Config) news.Fetcher { return NewLobsters(cfg) })
}

const lobstersURL = "https://lobste.rs/t/go.json"

// NewLobsters creates a Lobsters client targeting the Go tag.
func NewLobsters(_ env.Config) *Lobsters {
	return &Lobsters{url: lobstersURL}
}

// Fetch retrieves the latest Go-tagged stories from lobste.rs.
func (l Lobsters) Fetch(ctx context.Context) ([]news.Item, error) {
	stories, err := ingest.Fetch[[]lobstersStory](ctx, l.url, "lobsters", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, stories), nil
}

func (s lobstersStory) ShouldInclude() bool   { return true }
func (s lobstersStory) EnrichmentURL() string { return s.URL }

// Transform maps a lobstersStory to a news.Item.
//
// URL is the external story URL (the click target). OriginalURL is the
// Lobsters discussion page (comments_url) when it differs from the story URL.
// For Lobsters self-posts the two are identical, so OriginalURL stays empty.
func (s lobstersStory) Transform() news.Item {
	published, _ := time.Parse(time.RFC3339, s.CreatedAt)
	original := s.CommentsURL
	if original == s.URL {
		original = ""
	}
	return news.Item{
		Source:      news.SourceLobsters,
		Title:       s.Title,
		URL:         s.URL,
		OriginalURL: original,
		Author: &news.Author{
			Username:   s.SubmitterUser,
			ProfileURL: "https://lobste.rs/u/" + s.SubmitterUser,
		},
		Snippet:   s.Description,
		Tag:       news.TagDiscussion,
		Comments:  s.CommentCount,
		Score:     news.ScoreOf(news.SourceLobsters, news.TagDiscussion, float64(s.Score), true),
		Published: published.UTC(),
	}
}

type lobstersStory struct {
	ShortID       string   `json:"short_id"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	CommentsURL   string   `json:"comments_url"`
	Score         int      `json:"score"`
	CommentCount  int      `json:"comment_count"`
	CreatedAt     string   `json:"created_at"`
	Description   string   `json:"description"`
	SubmitterUser string   `json:"submitter_user"`
	Tags          []string `json:"tags"`
}
