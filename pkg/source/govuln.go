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
	"math"
	"sort"
	"strings"
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

// cvssScore returns the highest CVSS v3 base score found in the advisory's
// severity list. Returns (0, false) when no parseable CVSS v3 entry is present.
func (v vulnEntry) cvssScore() (float64, bool) {
	for _, s := range v.Severity {
		if s.Type == "CVSS_V3" {
			if score, ok := parseCVSSv3(s.Score); ok {
				return score, true
			}
		}
	}
	return 0, false
}

// parseCVSSv3 parses a CVSS v3.x vector string (e.g.
// "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H") and returns the base
// score in [0, 10] using the standard CVSS v3 formula.
func parseCVSSv3(vector string) (float64, bool) {
	const prefix = "CVSS:3."
	if !strings.HasPrefix(vector, prefix) {
		return 0, false
	}
	slash := strings.Index(vector[len(prefix):], "/")
	if slash < 0 {
		return 0, false
	}
	m := make(map[string]string, 8)
	for _, part := range strings.Split(vector[len(prefix)+slash+1:], "/") {
		k, val, ok := strings.Cut(part, ":")
		if !ok {
			return 0, false
		}
		m[k] = val
	}

	scope := m["S"]
	av, ok1 := cvssAV(m["AV"])
	ac, ok2 := cvssAC(m["AC"])
	pr, ok3 := cvssPR(m["PR"], scope)
	ui, ok4 := cvssUI(m["UI"])
	c, ok5 := cvssImpact(m["C"])
	i, ok6 := cvssImpact(m["I"])
	a, ok7 := cvssImpact(m["A"])
	if !(ok1 && ok2 && ok3 && ok4 && ok5 && ok6 && ok7) {
		return 0, false
	}

	iscBase := 1.0 - (1.0-c)*(1.0-i)*(1.0-a)
	var isc float64
	if scope == "U" {
		isc = 6.42 * iscBase
	} else {
		isc = 7.52*(iscBase-0.029) - 3.25*math.Pow(iscBase-0.02, 15)
	}
	if isc <= 0 {
		return 0, true
	}

	exploitability := 8.22 * av * ac * pr * ui
	var base float64
	if scope == "U" {
		base = math.Min(isc+exploitability, 10)
	} else {
		base = math.Min(1.08*(isc+exploitability), 10)
	}
	return math.Ceil(base*10) / 10, true
}

func cvssAV(v string) (float64, bool) {
	switch v {
	case "N":
		return 0.85, true
	case "A":
		return 0.62, true
	case "L":
		return 0.55, true
	case "P":
		return 0.20, true
	}
	return 0, false
}

func cvssAC(v string) (float64, bool) {
	switch v {
	case "L":
		return 0.77, true
	case "H":
		return 0.44, true
	}
	return 0, false
}

func cvssPR(v, scope string) (float64, bool) {
	switch v {
	case "N":
		return 0.85, true
	case "L":
		if scope == "C" {
			return 0.68, true
		}
		return 0.62, true
	case "H":
		if scope == "C" {
			return 0.50, true
		}
		return 0.27, true
	}
	return 0, false
}

func cvssUI(v string) (float64, bool) {
	switch v {
	case "N":
		return 0.85, true
	case "R":
		return 0.62, true
	}
	return 0, false
}

func cvssImpact(v string) (float64, bool) {
	switch v {
	case "N":
		return 0.00, true
	case "L":
		return 0.22, true
	case "H":
		return 0.56, true
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
