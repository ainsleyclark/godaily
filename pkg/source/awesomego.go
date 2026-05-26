// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// AwesomeGo defines the type that implements news.Fetcher for the
// avelino/awesome-go repository's commits feed — a curated stream of
// new packages and edits to the canonical Go list.
type AwesomeGo struct {
	url   string
	token string
}

var _ news.Fetcher = &AwesomeGo{}

func init() {
	news.Register(news.SourceAwesomeGo, func(cfg env.Config) news.Fetcher { return NewAwesomeGo(cfg) })
}

const awesomeGoURL = "https://api.github.com/repos/avelino/awesome-go/commits?per_page=20"

// NewAwesomeGo creates an Awesome Go commits client. It reuses cfg.GitHubToken
// when set so the source shares the elevated 5000/hr rate limit with the
// existing GitHub fetcher.
func NewAwesomeGo(cfg env.Config) *AwesomeGo {
	return &AwesomeGo{
		url:   awesomeGoURL,
		token: cfg.GitHubToken,
	}
}

// Fetch retrieves the latest commits to avelino/awesome-go.
func (a AwesomeGo) Fetch(ctx context.Context) ([]news.Item, error) {
	var hdrs []http.Header
	if a.token != "" {
		hdrs = []http.Header{{"Authorization": {"Bearer " + a.token}}}
	}
	commits, err := ingest.Fetch[[]awesomeCommit](ctx, a.url, "awesome go", json.Unmarshal, hdrs...)
	if err != nil {
		return nil, err
	}
	return ingest.TransformAll(ctx, commits), nil
}

// ShouldInclude drops merge commits, which are pure noise for a digest reader
// (they restate the PR title that the squashed commit already carried).
func (c awesomeCommit) ShouldInclude() bool {
	msg := c.Commit.Message
	return !strings.HasPrefix(msg, "Merge pull request") && !strings.HasPrefix(msg, "Merge branch")
}

// EnrichmentURL is empty: GitHub commit pages don't carry meaningful OG
// snippets or images, and the message itself is the content.
func (c awesomeCommit) EnrichmentURL() string { return "" }

// Transform maps an awesomeCommit to a news.Item. The first line of the
// commit message is the title; the rest (if any) becomes the snippet.
func (c awesomeCommit) Transform() news.Item {
	title, body := splitCommitMessage(c.Commit.Message)
	return news.Item{
		Source:    news.SourceAwesomeGo,
		Title:     title,
		URL:       c.HTMLURL,
		Author:    &news.Author{Name: c.Commit.Author.Name},
		Snippet:   body,
		Tag:       news.TagTrending,
		Score:     news.ScoreOf(news.SourceAwesomeGo, news.TagTrending, 0, false),
		Published: c.Commit.Author.Date,
	}
}

// splitCommitMessage splits a Git commit message into its first-line subject
// and remaining body. Returns (subject, body) with body empty when there is
// only a single line.
func splitCommitMessage(msg string) (string, string) {
	msg = strings.TrimSpace(msg)
	idx := strings.IndexByte(msg, '\n')
	if idx < 0 {
		return msg, ""
	}
	return strings.TrimSpace(msg[:idx]), strings.TrimSpace(msg[idx+1:])
}

type (
	awesomeCommit struct {
		SHA     string             `json:"sha"`
		HTMLURL string             `json:"html_url"`
		Commit  awesomeCommitInner `json:"commit"`
	}
	awesomeCommitInner struct {
		Message string              `json:"message"`
		Author  awesomeCommitAuthor `json:"author"`
	}
	awesomeCommitAuthor struct {
		Name string    `json:"name"`
		Date time.Time `json:"date"`
	}
)
