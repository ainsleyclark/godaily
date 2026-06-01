// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// Remotive fetches Go-relevant remote roles from the Remotive public jobs API.
// The API is unauthenticated and scoped to the software-development category;
// it is then filtered down to listings that mention Go in the title or tags.
// Remotive asks consumers to fetch no more than a few times a day (well within
// the once-per-window collection) and to credit it as the source, which the
// NiceName / link in the rendered digest does.
type Remotive struct {
	url string
	now func() time.Time
}

var _ news.Fetcher = &Remotive{}

func init() {
	news.Register(news.SourceRemotive, func(cfg env.Config) news.Fetcher { return NewRemotive(cfg) })
}

const remotiveURL = "https://remotive.com/api/remote-jobs?category=software-dev"

// NewRemotive creates a Remotive client scoped to software-development roles.
func NewRemotive(_ env.Config) *Remotive {
	return &Remotive{url: remotiveURL, now: time.Now}
}

// Fetch retrieves software-development roles from Remotive and returns the
// Go-relevant ones as news items.
func (r Remotive) Fetch(ctx context.Context) ([]news.Item, error) {
	headers := http.Header{"User-Agent": []string{"godaily/1.0 (+https://godaily.dev)"}}
	resp, err := ingest.Fetch[remotiveResponse](ctx, r.url, "remotive", json.Unmarshal, headers)
	if err != nil {
		return nil, err
	}
	now := r.now().UTC()
	for i := range resp.Jobs {
		resp.Jobs[i].ageDays = remotiveAgeDays(now, resp.Jobs[i].PublicationDate)
		resp.Jobs[i].now = now
	}
	return ingest.TransformAll(ctx, resp.Jobs), nil
}

// remotivePubLayout is the timestamp format Remotive returns, e.g.
// "2024-12-31 10:23:26". The API omits a timezone; values are UTC.
const remotivePubLayout = "2006-01-02 15:04:05"

// remotiveAgeDays parses Remotive's publication timestamp and returns whole days
// elapsed since now, floored at zero. Unparseable or future dates yield zero so
// they neither error nor earn a runaway recency boost.
func remotiveAgeDays(now time.Time, published string) int {
	posted, err := time.Parse(remotivePubLayout, published)
	if err != nil {
		posted, err = time.Parse(time.RFC3339, published)
		if err != nil {
			return 0
		}
	}
	days := int(now.Sub(posted.UTC()).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func (j remotiveJob) ShouldInclude() bool {
	if j.Title == "" || j.URL == "" {
		return false
	}
	return hasGoWord(j.Title) || tagsContainGo(j.Tags)
}

func (j remotiveJob) EnrichmentURL() string { return "" }

func (j remotiveJob) Transform() news.Item {
	goTitle := hasGoWord(j.Title)
	salary := hasSalary(j.Salary)
	// Remotive is a remote-only board; a blank location means remote-anywhere.
	remote := j.Location == "" || isRemote(j.Location)

	weight := news.SourceWeight(news.SourceRemotive, news.TagJobs)
	score := weight * news.JobBoost(j.ageDays, goTitle, salary, remote)

	var author *news.Author
	if j.Company != "" {
		author = &news.Author{Name: j.Company}
	}

	return news.Item{
		Source:    news.SourceRemotive,
		Title:     buildJobTitle(j.Company, j.Title, remotiveLocation(j.Location)),
		URL:       j.URL,
		Author:    author,
		Snippet:   j.Salary, // surfaced only when the listing discloses a range
		Tag:       news.TagJobs,
		Score:     score,
		Published: j.now,
	}
}

// remotiveLocation falls back to "Remote" when the listing leaves the required
// location blank, matching the Remote OK convention for the remote-only board.
func remotiveLocation(s string) string {
	if s == "" {
		return "Remote"
	}
	return s
}

type (
	remotiveResponse struct {
		Jobs []remotiveJob `json:"jobs"`
	}
	remotiveJob struct {
		ID              int      `json:"id"`
		URL             string   `json:"url"`
		Title           string   `json:"title"`
		Company         string   `json:"company_name"`
		Location        string   `json:"candidate_required_location"`
		Salary          string   `json:"salary"`
		PublicationDate string   `json:"publication_date"`
		Tags            []string `json:"tags"`

		ageDays int       // populated by Fetch before TransformAll runs
		now     time.Time // snapshot of collection time, used as Published
	}
)
