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
	"sort"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// GoVuln fetches recent Go vulnerability advisories from the Go vulnerability
// database (vuln.go.dev) and surfaces them as security news items.
type GoVuln struct {
	indexURL   string
	detailBase string
	window     time.Duration
	limit      int
}

var _ news.Fetcher = &GoVuln{}

func init() {
	news.Register(news.SourceGoVuln, func(_ env.Config) news.Fetcher { return NewGoVuln() })
}

const (
	goVulnIndexURL   = "https://vuln.go.dev/index/vulns.json"
	goVulnDetailBase = "https://vuln.go.dev/ID/"
	goVulnWindow     = 7 * 24 * time.Hour
	goVulnLimit      = 10
)

// NewGoVuln returns a GoVuln fetcher configured against the production
// vulnerability database.
func NewGoVuln() *GoVuln {
	return &GoVuln{
		indexURL:   goVulnIndexURL,
		detailBase: goVulnDetailBase,
		window:     goVulnWindow,
		limit:      goVulnLimit,
	}
}

// Fetch retrieves the vulnerability index, filters to entries modified within
// the configured window, then fetches OSV details for each. Errors on
// individual detail requests are non-fatal — that entry is silently skipped so
// a transient failure does not discard the whole batch.
func (g GoVuln) Fetch(ctx context.Context) ([]news.Item, error) {
	index, err := ingest.Fetch[[]vulnIndexEntry](ctx, g.indexURL, "go vuln index", json.Unmarshal)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-g.window)
	var recent []vulnIndexEntry
	for _, e := range index {
		if e.Modified.After(cutoff) {
			recent = append(recent, e)
		}
	}

	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Modified.After(recent[j].Modified)
	})
	if g.limit > 0 && len(recent) > g.limit {
		recent = recent[:g.limit]
	}

	var entries []vulnEntry
	for _, e := range recent {
		detail, detailErr := ingest.Fetch[vulnEntry](ctx, g.detailBase+e.ID+".json", "go vuln detail", json.Unmarshal)
		if detailErr != nil {
			continue
		}
		entries = append(entries, detail)
	}

	return ingest.TransformAll(ctx, entries), nil
}

func (v vulnEntry) ShouldInclude() bool   { return v.Withdrawn == nil && v.Summary != "" }
func (v vulnEntry) EnrichmentURL() string { return "" }

func (v vulnEntry) Transform() news.Item {
	return news.Item{
		Source:    news.SourceGoVuln,
		Title:     v.Summary,
		URL:       "https://pkg.go.dev/vuln/" + v.ID,
		Snippet:   v.Details,
		Tag:       news.TagSecurity,
		Score:     news.ScoreOf(news.SourceGoVuln, news.TagSecurity, 0, false),
		Published: v.Published,
	}
}

type (
	vulnIndexEntry struct {
		ID       string    `json:"id"`
		Modified time.Time `json:"modified"`
	}

	vulnEntry struct {
		ID        string         `json:"id"`
		Published time.Time      `json:"published"`
		Summary   string         `json:"summary"`
		Details   string         `json:"details"`
		Affected  []vulnAffected `json:"affected"`
		Withdrawn *time.Time     `json:"withdrawn,omitempty"`
	}

	vulnAffected struct {
		Package vulnPackage `json:"package"`
	}

	vulnPackage struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	}
)
