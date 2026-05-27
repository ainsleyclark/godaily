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
	"net/http"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// RemoteOK fetches Go-tagged remote job listings from remoteok.com.
// The API is unauthenticated but blocks the default Go user agent, so a
// custom one is sent on every request.
type RemoteOK struct {
	url string
	now func() time.Time
}

var _ news.Fetcher = &RemoteOK{}

func init() {
	news.Register(news.SourceRemoteOK, func(cfg env.Config) news.Fetcher { return NewRemoteOK(cfg) })
}

const remoteOKURL = "https://remoteok.com/api?tags=golang"

// NewRemoteOK creates a Remote OK client.
func NewRemoteOK(_ env.Config) *RemoteOK {
	return &RemoteOK{url: remoteOKURL, now: time.Now}
}

// Fetch retrieves Go-tagged jobs from Remote OK. The API returns a
// heterogeneous JSON array whose first element is a legal/metadata object;
// it is filtered out by ShouldInclude (Position will be empty).
func (r RemoteOK) Fetch(ctx context.Context) ([]news.Item, error) {
	headers := http.Header{"User-Agent": []string{"godaily/1.0 (+https://godaily.dev)"}}
	jobs, err := ingest.Fetch[[]remoteOKJob](ctx, r.url, "remote ok", json.Unmarshal, headers)
	if err != nil {
		return nil, err
	}
	now := r.now().UTC()
	for i := range jobs {
		jobs[i].ageDays = remoteOKAgeDays(now, jobs[i].Epoch)
	}
	return ingest.TransformAll(ctx, jobs), nil
}

// remoteOKAgeDays converts a Unix epoch into whole days elapsed since now,
// floored at zero so future-dated listings (which we have seen on the API
// for jobs posted "next Monday") don't get a runaway recency boost.
func remoteOKAgeDays(now time.Time, epoch int64) int {
	if epoch <= 0 {
		return 0
	}
	posted := time.Unix(epoch, 0).UTC()
	days := int(now.Sub(posted).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// remoteOKMaxTags caps the tag count for tag-only matches. Legitimate Go
// listings on Remote OK have ~4 tags (golang, senior, backend, dev);
// keyword-fishing spam carries 13+ across unrelated industries (medical,
// recruiter, marketing, finance, ...). The gap is wide enough to draw a
// clean line at 10.
const remoteOKMaxTags = 10

func (j remoteOKJob) ShouldInclude() bool {
	if j.Position == "" || j.URL == "" {
		return false
	}
	if hasGoWord(j.Position) {
		return true
	}
	// Tag-only match: only trust the listing when its tag set isn't a
	// keyword-fishing pile.
	return len(j.Tags) <= remoteOKMaxTags && tagsContainGo(j.Tags)
}

func (j remoteOKJob) EnrichmentURL() string { return "" }

func (j remoteOKJob) Transform() news.Item {
	salary := j.SalaryMin > 0 || j.SalaryMax > 0
	remote := true // Remote OK is remote-only by definition.
	goTitle := hasGoWord(j.Position)

	weight := news.SourceWeight(news.SourceRemoteOK, news.TagJobs)
	score := weight * news.JobBoost(j.ageDays, goTitle, salary, remote)

	target := j.ApplyURL
	if target == "" {
		target = j.URL
	}

	var author *news.Author
	if j.Company != "" {
		author = &news.Author{Name: j.Company}
	}

	published := time.Time{}
	if j.Epoch > 0 {
		published = time.Unix(j.Epoch, 0).UTC()
	}

	return news.Item{
		Source:      news.SourceRemoteOK,
		Title:       j.Position,
		URL:         target,
		OriginalURL: j.URL,
		Author:      author,
		Snippet:     buildRemoteOKSnippet(j),
		Tag:         news.TagJobs,
		Score:       score,
		Published:   published,
	}
}

// buildRemoteOKSnippet composes a short meta line: company · location · salary.
// Description is intentionally omitted — it's frequently a wall of HTML and
// the trio above is the highest-signal context for a daily digest.
func buildRemoteOKSnippet(j remoteOKJob) string {
	var parts []string
	if j.Company != "" {
		parts = append(parts, j.Company)
	}
	// The API regularly returns half-typed locations such as "Reston, " or
	// "London, UK,". Trim trailing punctuation so the snippet doesn't end
	// mid-thought.
	if loc := strings.Trim(j.Location, " ,\t\n"); loc != "" {
		parts = append(parts, loc)
	} else {
		parts = append(parts, "Remote")
	}
	if s := formatSalary(j.SalaryMin, j.SalaryMax); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, " · ")
}

// formatSalary renders the disclosed salary range as e.g. "$80k–$120k" or
// "$80k+" / "up to $120k" when only one bound is set. Returns "" when both
// bounds are missing.
func formatSalary(minVal, maxVal float64) string {
	switch {
	case minVal > 0 && maxVal > 0:
		return fmt.Sprintf("$%s–$%s", formatThousands(minVal), formatThousands(maxVal))
	case minVal > 0:
		return fmt.Sprintf("$%s+", formatThousands(minVal))
	case maxVal > 0:
		return fmt.Sprintf("up to $%s", formatThousands(maxVal))
	default:
		return ""
	}
}

func formatThousands(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.0fk", v/1000)
	}
	return fmt.Sprintf("%.0f", v)
}

func tagsContainGo(tags []string) bool {
	for _, t := range tags {
		if hasGoWord(t) {
			return true
		}
	}
	return false
}

type remoteOKJob struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Epoch       int64    `json:"epoch"`
	Company     string   `json:"company"`
	Position    string   `json:"position"`
	Tags        []string `json:"tags"`
	Location    string   `json:"location"`
	SalaryMin   float64  `json:"salary_min"`
	SalaryMax   float64  `json:"salary_max"`
	ApplyURL    string   `json:"apply_url"`
	URL         string   `json:"url"`
	Description string   `json:"description"`

	ageDays int // populated by Fetch before TransformAll runs
}
