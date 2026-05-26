// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	cvss30 "github.com/pandatix/go-cvss/30"
	cvss31 "github.com/pandatix/go-cvss/31"

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
	cvss, hasCVSS := v.cvssScore()
	return news.Item{
		Source:    news.SourceGoVuln,
		Title:     v.Summary,
		URL:       "https://pkg.go.dev/vuln/" + v.ID,
		Snippet:   v.Details,
		Tag:       news.TagSecurity,
		Score:     news.ScoreOf(news.SourceGoVuln, news.TagSecurity, cvss, hasCVSS),
		Published: v.Modified,
	}
}

// cvssScore returns the CVSS v3 base score for the advisory, trying v3.1
// then v3.0. Returns (0, false) when no parseable CVSS v3 entry is present.
func (v vulnEntry) cvssScore() (float64, bool) {
	for _, s := range v.Severity {
		if s.Type != "CVSS_V3" {
			continue
		}
		if cvss, err := cvss31.ParseVector(s.Score); err == nil {
			return cvss.BaseScore(), true
		}
		if cvss, err := cvss30.ParseVector(s.Score); err == nil {
			return cvss.BaseScore(), true
		}
	}
	return 0, false
}

type (
	vulnIndexEntry struct {
		ID       string    `json:"id"`
		Modified time.Time `json:"modified"`
	}

	vulnEntry struct {
		ID        string         `json:"id"`
		Published time.Time      `json:"published"`
		Modified  time.Time      `json:"modified"`
		Summary   string         `json:"summary"`
		Details   string         `json:"details"`
		Severity  []vulnSeverity `json:"severity"`
		Affected  []vulnAffected `json:"affected"`
		Withdrawn *time.Time     `json:"withdrawn,omitempty"`
	}

	vulnSeverity struct {
		Type  string `json:"type"`
		Score string `json:"score"` // CVSS vector string, e.g. "CVSS:3.1/AV:N/..."
	}

	vulnAffected struct {
		Package vulnPackage `json:"package"`
	}

	vulnPackage struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	}
)
