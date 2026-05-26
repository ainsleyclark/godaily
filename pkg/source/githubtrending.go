// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
	"github.com/pkg/errors"
)

// GitHubTrending scrapes github.com/trending/go since the page has no JSON API.
type GitHubTrending struct {
	url string
}

var _ news.Fetcher = &GitHubTrending{}

func init() {
	news.Register(news.SourceGitHubTrending, func(cfg env.Config) news.Fetcher { return NewGitHubTrending(cfg) })
}

const githubTrendingURL = "https://github.com/trending/go?since=daily"

// NewGitHubTrending creates a GitHub Trending (Go) scraper.
func NewGitHubTrending(_ env.Config) *GitHubTrending {
	return &GitHubTrending{url: githubTrendingURL}
}

// Fetch retrieves the daily trending Go repositories. The page has no per-card
// timestamp; "stars today" is a rolling 24h window, so every item is stamped
// with yesterday-noon UTC to land inside the digest's accept window.
func (g GitHubTrending) Fetch(ctx context.Context) ([]news.Item, error) {
	doc, err := ingest.FetchHTML(ctx, g.url, "github trending")
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(g.url)
	if err != nil {
		return nil, errors.Wrap(err, "github trending: parsing base url")
	}
	hostPrefix := base.Scheme + "://" + base.Host

	publishedAt := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(12 * time.Hour)

	var repos []trendingRepo
	doc.Find("article.Box-row").Each(func(_ int, s *goquery.Selection) {
		if r, ok := parseTrendingCard(s, hostPrefix, publishedAt); ok {
			repos = append(repos, r)
		}
	})
	return ingest.TransformAll(ctx, repos), nil
}

type trendingRepo struct {
	Title       string
	URL         string
	Author      string
	Description string
	StarsToday  int
	Published   time.Time
}

func (r trendingRepo) ShouldInclude() bool   { return r.URL != "" }
func (r trendingRepo) EnrichmentURL() string { return r.URL }

func (r trendingRepo) Transform() news.Item {
	return news.Item{
		Source: news.SourceGitHubTrending,
		Title:  r.Title,
		URL:    r.URL,
		Author: &news.Author{
			Username:   r.Author,
			ProfileURL: "https://github.com/" + r.Author,
		},
		Snippet:   r.Description,
		Tag:       news.TagTrending,
		Score:     news.ScoreOf(news.SourceGitHubTrending, news.TagTrending, float64(r.StarsToday), true),
		Published: r.Published,
	}
}

// parseTrendingCard pulls fields from a single <article.Box-row>; returns
// ok=false when the card has no usable repo link so callers can skip it
// rather than aborting the whole fetch on one malformed entry.
func parseTrendingCard(s *goquery.Selection, hostPrefix string, published time.Time) (trendingRepo, bool) {
	href, _ := s.Find("h2 a").First().Attr("href")
	href = strings.TrimSpace(href)
	if href == "" {
		return trendingRepo{}, false
	}

	// href is always "/owner/repo" — split off the owner for Author while
	// keeping the full "owner/repo" as Title so the digest still shows the
	// fully-qualified repo identifier.
	path := strings.TrimPrefix(href, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return trendingRepo{}, false
	}
	author := parts[0]
	title := path
	desc := strings.TrimSpace(s.Find("p").First().Text())

	starsToday := 0
	if raw := s.Find("span.d-inline-block.float-sm-right").First().Text(); raw != "" {
		starsToday, _ = strconv.Atoi(strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, raw))
	}

	return trendingRepo{
		Title:       title,
		URL:         hostPrefix + href,
		Author:      author,
		Description: desc,
		StarsToday:  starsToday,
		Published:   published,
	}, true
}
