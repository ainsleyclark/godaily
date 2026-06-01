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

// WeWorkRemotely fetches remote programming jobs from We Work Remotely's RSS
// feed and keeps the ones that mention Go. The category feed is not Go-specific
// (it covers the whole programming category), so listings are filtered on a
// whole-word Go match in the title. A custom User-Agent is sent because the
// default Go agent is blocked.
type WeWorkRemotely struct {
	url string
	now func() time.Time
}

var _ news.Fetcher = &WeWorkRemotely{}

func init() {
	news.Register(news.SourceWeWorkRemotely, func(cfg env.Config) news.Fetcher { return NewWeWorkRemotely(cfg) })
}

const weWorkRemotelyURL = "https://weworkremotely.com/categories/remote-programming-jobs.rss"

// NewWeWorkRemotely creates a We Work Remotely client scoped to the remote
// programming-jobs category feed.
func NewWeWorkRemotely(_ env.Config) *WeWorkRemotely {
	return &WeWorkRemotely{url: weWorkRemotelyURL, now: time.Now}
}

// Fetch retrieves the programming-jobs feed and returns the Go-relevant
// listings as news items.
func (w WeWorkRemotely) Fetch(ctx context.Context) ([]news.Item, error) {
	headers := http.Header{"User-Agent": []string{"godaily/1.0 (+https://godaily.dev)"}}
	feed, err := ingest.Fetch[wwrFeed](ctx, w.url, "we work remotely", xml.Unmarshal, headers)
	if err != nil {
		return nil, err
	}
	now := w.now().UTC()
	for i := range feed.Channel.Items {
		feed.Channel.Items[i].now = now
	}
	return ingest.TransformAll(ctx, feed.Channel.Items), nil
}

func (i wwrItem) ShouldInclude() bool {
	// Match Go only in the title. The programming-category feed isn't
	// Go-specific, and matching the description lets the common English word
	// "go" in marketing copy ("ready to go", "go further") drag in unrelated
	// roles — every one of a sample run's hits was a false positive.
	return strings.TrimSpace(i.Link) != "" && hasGoWord(i.Title)
}

func (i wwrItem) EnrichmentURL() string { return i.Link }

func (i wwrItem) Transform() news.Item {
	company, role := wwrCompanyRole(i.Title)

	goTitle := hasGoWord(role)
	salary := hasSalary(i.Description)
	remote := true // We Work Remotely is a remote-only board.

	weight := news.SourceWeight(news.SourceWeWorkRemotely, news.TagJobs)
	score := weight * news.JobBoost(jobFeedAgeDays(i.now, i.PubDate), goTitle, salary, remote)

	var author *news.Author
	if company != "" {
		author = &news.Author{Name: company}
	}

	return news.Item{
		Source:    news.SourceWeWorkRemotely,
		Title:     buildJobTitle(company, role, ""),
		URL:       i.Link,
		Author:    author,
		Snippet:   i.Description,
		Tag:       news.TagJobs,
		Score:     score,
		Published: i.now,
	}
}

// wwrCompanyRole splits We Work Remotely's "Company: Role" title convention into
// its two parts. Falls back to an empty company and the whole title as the role
// when the colon convention isn't followed.
func wwrCompanyRole(title string) (company, role string) {
	title = strings.TrimSpace(title)
	if i := strings.Index(title, ":"); i > 0 {
		return strings.TrimSpace(title[:i]), strings.TrimSpace(title[i+1:])
	}
	return "", title
}

type (
	wwrFeed struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Items []wwrItem `xml:"item"`
		} `xml:"channel"`
	}
	wwrItem struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`

		now time.Time // snapshot of collection time, used as Published
	}
)
