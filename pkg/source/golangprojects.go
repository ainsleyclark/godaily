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

// GolangProjects fetches Go roles from the Golangprojects job board RSS feed.
// The board is Go-only, so every listing is relevant and no Go keyword filter
// is applied — items are kept on a non-empty link.
type GolangProjects struct {
	url string
	now func() time.Time
}

var _ news.Fetcher = &GolangProjects{}

func init() {
	news.Register(news.SourceGolangProjects, func(cfg env.Config) news.Fetcher { return NewGolangProjects(cfg) })
}

const golangProjectsURL = "https://www.golangprojects.com/rss/jobs.xml"

// NewGolangProjects creates a Golangprojects RSS client.
func NewGolangProjects(_ env.Config) *GolangProjects {
	return &GolangProjects{url: golangProjectsURL, now: time.Now}
}

// Fetch retrieves Go roles from the Golangprojects RSS feed.
func (g GolangProjects) Fetch(ctx context.Context) ([]news.Item, error) {
	headers := http.Header{"User-Agent": []string{"godaily/1.0 (+https://godaily.dev)"}}
	feed, err := ingest.Fetch[golangProjectsFeed](ctx, g.url, "golang projects", xml.Unmarshal, headers)
	if err != nil {
		return nil, err
	}
	now := g.now().UTC()
	for i := range feed.Channel.Items {
		feed.Channel.Items[i].now = now
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (i golangProjectsItem) ShouldInclude() bool   { return strings.TrimSpace(i.Link) != "" }
func (i golangProjectsItem) EnrichmentURL() string { return i.Link }

func (i golangProjectsItem) Transform() news.Item {
	company, role := jobRoleAt(i.Title)

	salary := hasSalary(i.Title) || hasSalary(i.Description)
	remote := isRemote(i.Title) || isRemote(i.Description)
	// Go-only board: treat every role as Go-relevant for the boost.
	weight := news.SourceWeight(news.SourceGolangProjects, news.TagJobs)
	score := weight * news.JobBoost(jobFeedAgeDays(i.now, i.PubDate), true, salary, remote)

	var author *news.Author
	if company != "" {
		author = &news.Author{Name: company}
	}

	return news.Item{
		Source:    news.SourceGolangProjects,
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
	golangProjectsFeed struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Items []golangProjectsItem `xml:"item"`
		} `xml:"channel"`
	}
	golangProjectsItem struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`

		now time.Time // snapshot of collection time, used as Published
	}
)
