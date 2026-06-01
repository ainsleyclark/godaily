// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// GolangCafe fetches Go roles from the Golang.cafe job board RSS feed. The board
// is Go-only (no recruiters, clear salary ranges), so every listing is relevant
// and no Go keyword filter is applied — items are kept on a non-empty link.
//
// Golang.cafe sits behind Cloudflare and 429s direct datacenter requests, so
// when ScraperAPI keys are configured the feed is fetched through the standard
// (non-premium) proxy pool with the request headers forwarded. Without keys it
// falls back to a direct request, which works locally but is liable to 429.
type GolangCafe struct {
	url string
	now func() time.Time
}

var _ news.Fetcher = &GolangCafe{}

func init() {
	news.Register(news.SourceGolangCafe, func(cfg env.Config) news.Fetcher { return NewGolangCafe(cfg) })
}

const golangCafeURL = "https://golang.cafe/rss"

// NewGolangCafe creates a Golang.cafe RSS client, proxying through ScraperAPI
// (standard pool) when keys are available to clear Cloudflare.
func NewGolangCafe(cfg env.Config) *GolangCafe {
	url := ingest.ScraperURL(cfg.ScraperAPIKeys, golangCafeURL, ingest.WithKeepHeaders(), ingest.WithoutPremium())
	return &GolangCafe{url: url, now: time.Now}
}

// Fetch retrieves Go roles from the Golang.cafe RSS feed.
func (g GolangCafe) Fetch(ctx context.Context) ([]news.Item, error) {
	headers := http.Header{"User-Agent": []string{"godaily/1.0 (+https://godaily.dev)"}}
	feed, err := ingest.Fetch[golangCafeFeed](ctx, g.url, "golang cafe", xml.Unmarshal, headers)
	if err != nil {
		return nil, err
	}
	now := g.now().UTC()
	for i := range feed.Channel.Items {
		feed.Channel.Items[i].now = now
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (i golangCafeItem) ShouldInclude() bool   { return strings.TrimSpace(i.Link) != "" }
func (i golangCafeItem) EnrichmentURL() string { return i.Link }

func (i golangCafeItem) Transform() news.Item {
	company, role := jobRoleAt(i.Title)

	salary := hasSalary(i.Title) || hasSalary(i.Description)
	remote := isRemote(i.Title) || isRemote(i.Description)
	// Go-only board: the role is Go even when the title doesn't spell it out.
	weight := news.SourceWeight(news.SourceGolangCafe, news.TagJobs)
	score := weight * news.JobBoost(jobFeedAgeDays(i.now, i.PubDate), true, salary, remote)

	var author *news.Author
	if company != "" {
		author = &news.Author{Name: company}
	}

	return news.Item{
		Source:    news.SourceGolangCafe,
		Title:     buildJobTitle(company, role, ""),
		URL:       i.Link,
		Author:    author,
		Snippet:   i.Description,
		Tag:       news.TagJobs,
		Score:     score,
		Published: i.now,
	}
}

type (
	golangCafeFeed struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Items []golangCafeItem `xml:"item"`
		} `xml:"channel"`
	}
	golangCafeItem struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`

		now time.Time // snapshot of collection time, used as Published
	}
)
