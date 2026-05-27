// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
		Title:       buildRemoteOKTitle(j),
		URL:         target,
		OriginalURL: j.URL,
		Author:      author,
		Snippet:     buildRemoteOKSnippet(j),
		Tag:         news.TagJobs,
		Score:       score,
		Published:   published,
	}
}

// buildRemoteOKTitle composes the link text as "Company · Role · Location".
// Putting the employer first mirrors the HN whoishiring convention and gives
// the otherwise-bare title some weight; without it readers see only a role
// and have to scan the snippet for context. Falls back gracefully when any
// field is missing.
func buildRemoteOKTitle(j remoteOKJob) string {
	if j.Position == "" {
		return ""
	}
	parts := make([]string, 0, 3)
	if j.Company != "" {
		parts = append(parts, j.Company)
	}
	parts = append(parts, j.Position)
	if loc := remoteOKDisplayLocation(j.Location); loc != "" {
		parts = append(parts, loc)
	}
	return strings.Join(parts, " · ")
}

// remoteOKDisplayLocation normalises the API's location field for display.
// Trims the dangling ", " the API regularly returns, maps the Portuguese
// "Remoto" to "Remote" for consistency, and falls back to "Remote" when the
// field is blank (Remote OK is a remote-only board, so the absence of a
// location means remote-anywhere rather than missing data).
func remoteOKDisplayLocation(s string) string {
	loc := strings.Trim(s, " ,\t\n")
	if loc == "" || strings.EqualFold(loc, "remoto") {
		return "Remote"
	}
	return loc
}

// buildRemoteOKSnippet returns the salary range, if disclosed. Company and
// location both live in the title now (see buildRemoteOKTitle), so the
// snippet's only remaining job is surfacing the comp band — the one piece
// of context worth a second line. Empty when no salary is on the listing,
// which the email template skips silently.
func buildRemoteOKSnippet(j remoteOKJob) string {
	return formatSalary(j.SalaryMin, j.SalaryMax)
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
