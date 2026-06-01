// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// GolangCafe scrapes the Golang.cafe job board. The site sits behind Cloudflare
// and 404s its /rss feed to datacenter proxies, but the listing page renders,
// and — being SEO-tuned for Google Jobs — embeds a schema.org JobPosting
// JSON-LD block for each listing. We parse those structured blocks rather than
// brittle markup. The board is Go-only, so every posting is relevant.
//
// ScraperAPI is used WITHOUT keep_headers so the proxy presents its own
// browser-like identity to Cloudflare (forwarding our godaily User-Agent is
// what the /rss feed rejected). Without keys it falls back to a direct request,
// which Cloudflare is liable to block.
type GolangCafe struct {
	url string
	now func() time.Time
}

var _ news.Fetcher = &GolangCafe{}

func init() {
	news.Register(news.SourceGolangCafe, func(cfg env.Config) news.Fetcher { return NewGolangCafe(cfg) })
}

const golangCafeURL = "https://golang.cafe"

// NewGolangCafe creates a Golang.cafe scraper, proxying through ScraperAPI
// (standard pool) when keys are available to clear Cloudflare.
func NewGolangCafe(cfg env.Config) *GolangCafe {
	return &GolangCafe{
		url: ingest.ScraperURL(cfg.ScraperAPIKeys, golangCafeURL, ingest.WithoutPremium()),
		now: time.Now,
	}
}

// Fetch scrapes the listing page and returns each JobPosting JSON-LD block as a
// news item.
func (g GolangCafe) Fetch(ctx context.Context) ([]news.Item, error) {
	doc, err := ingest.FetchHTML(ctx, g.url, "golang cafe")
	if err != nil {
		return nil, err
	}
	now := g.now().UTC()
	var jobs []golangCafeJob
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, s *goquery.Selection) {
		for _, jp := range jobPostingsFromJSONLD([]byte(s.Text())) {
			jobs = append(jobs, newGolangCafeJob(jp, now))
		}
	})
	return ingest.TransformAll(ctx, jobs), nil
}

type golangCafeJob struct {
	title    string
	url      string
	company  string
	location string
	salary   string
	remote   bool
	ageDays  int
	now      time.Time
}

func (j golangCafeJob) ShouldInclude() bool   { return j.title != "" && j.url != "" }
func (j golangCafeJob) EnrichmentURL() string { return "" }

func (j golangCafeJob) Transform() news.Item {
	weight := news.SourceWeight(news.SourceGolangCafe, news.TagJobs)
	// Go-only board: treat every posting as Go-relevant for the boost.
	score := weight * news.JobBoost(j.ageDays, true, j.salary != "", j.remote)

	var author *news.Author
	if j.company != "" {
		author = &news.Author{Name: j.company}
	}

	return news.Item{
		Source:    news.SourceGolangCafe,
		Title:     buildJobTitle(j.company, j.title, j.location),
		URL:       j.url,
		Author:    author,
		Snippet:   j.salary, // golang.cafe always discloses a range
		Tag:       news.TagJobs,
		Score:     score,
		Published: j.now,
	}
}

// newGolangCafeJob projects a parsed JobPosting object onto our intermediate
// type, deriving remote status from the location, jobLocationType, and the
// presence of applicantLocationRequirements.
func newGolangCafeJob(jp map[string]any, now time.Time) golangCafeJob {
	location := jsonLDLocation(jp["jobLocation"])
	remote := isRemote(location) ||
		strings.EqualFold(jsonLDString(jp["jobLocationType"]), "TELECOMMUTE") ||
		jp["applicantLocationRequirements"] != nil
	if location == "" && remote {
		location = "Remote"
	}
	return golangCafeJob{
		title:    strings.TrimSpace(jsonLDString(jp["title"])),
		url:      strings.TrimSpace(jsonLDString(jp["url"])),
		company:  jsonLDCompany(jp["hiringOrganization"]),
		location: location,
		salary:   jsonLDSalary(jp["baseSalary"]),
		remote:   remote,
		ageDays:  jsonLDAgeDays(now, jsonLDString(jp["datePosted"])),
		now:      now,
	}
}

// jsonLDAgeDays parses a JobPosting datePosted ("2006-01-02" or RFC3339) into
// whole days since now, floored at zero.
func jsonLDAgeDays(now time.Time, datePosted string) int {
	datePosted = strings.TrimSpace(datePosted)
	if datePosted == "" {
		return 0
	}
	posted, err := time.Parse("2006-01-02", datePosted)
	if err != nil {
		posted, err = time.Parse(time.RFC3339, datePosted)
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

// jobPostingsFromJSONLD parses a JSON-LD script body and returns every
// schema.org JobPosting object found anywhere within it — handling a top-level
// object or array, an @graph wrapper, or an ItemList's nested items.
func jobPostingsFromJSONLD(raw []byte) []map[string]any {
	var root any
	if json.Unmarshal(raw, &root) != nil {
		return nil
	}
	var out []map[string]any
	var walk func(n any)
	walk = func(n any) {
		switch t := n.(type) {
		case map[string]any:
			if jsonLDTypeIs(t["@type"], "JobPosting") {
				out = append(out, t)
			}
			for _, v := range t {
				walk(v)
			}
		case []any:
			for _, v := range t {
				walk(v)
			}
		}
	}
	walk(root)
	return out
}

// jsonLDTypeIs reports whether a JSON-LD @type (a string or array of strings)
// contains want, case-insensitively.
func jsonLDTypeIs(v any, want string) bool {
	switch t := v.(type) {
	case string:
		return strings.EqualFold(t, want)
	case []any:
		for _, e := range t {
			if s, ok := e.(string); ok && strings.EqualFold(s, want) {
				return true
			}
		}
	}
	return false
}

func jsonLDString(v any) string {
	s, _ := v.(string)
	return s
}

// jsonLDFirstMap returns v as a map, or the first map element when v is an array
// (JSON-LD fields are routinely either a single object or a list of them).
func jsonLDFirstMap(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return t
	case []any:
		for _, e := range t {
			if m, ok := e.(map[string]any); ok {
				return m
			}
		}
	}
	return nil
}

// jsonLDCompany reads hiringOrganization, which may be a bare string or an
// Organization object with a name.
func jsonLDCompany(v any) string {
	if s := strings.TrimSpace(jsonLDString(v)); s != "" {
		return s
	}
	if m := jsonLDFirstMap(v); m != nil {
		return strings.TrimSpace(jsonLDString(m["name"]))
	}
	return ""
}

// jsonLDLocation reads jobLocation, preferring the most specific address part
// available (locality, then region, then country).
func jsonLDLocation(v any) string {
	m := jsonLDFirstMap(v)
	if m == nil {
		return ""
	}
	addr := jsonLDFirstMap(m["address"])
	if addr == nil {
		return strings.TrimSpace(jsonLDString(m["name"]))
	}
	for _, k := range []string{"addressLocality", "addressRegion", "addressCountry"} {
		if s := strings.TrimSpace(jsonLDString(addr[k])); s != "" {
			return s
		}
	}
	return ""
}

// jsonLDSalary renders a baseSalary MonetaryAmount as e.g. "$120k–$160k", or a
// single-bound variant, or "" when no numeric range is present.
func jsonLDSalary(v any) string {
	m := jsonLDFirstMap(v)
	if m == nil {
		return ""
	}
	val := jsonLDFirstMap(m["value"])
	if val == nil {
		return ""
	}
	minV := jsonLDNumber(val["minValue"])
	maxV := jsonLDNumber(val["maxValue"])
	sym := currencySymbol(jsonLDString(m["currency"]))
	switch {
	case minV > 0 && maxV > 0:
		return sym + formatThousands(minV) + "–" + sym + formatThousands(maxV)
	case minV > 0:
		return sym + formatThousands(minV) + "+"
	case maxV > 0:
		return "up to " + sym + formatThousands(maxV)
	default:
		return ""
	}
}

// jsonLDNumber coerces a JSON-LD numeric field, which may arrive as a number or
// a comma-formatted string.
func jsonLDNumber(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		f, _ := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(t), ",", ""), 64)
		return f
	}
	return 0
}

// currencySymbol maps an ISO currency code to its symbol, defaulting to "$" when
// absent and falling back to the code itself for anything unmapped.
func currencySymbol(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "USD", "":
		return "$"
	case "GBP":
		return "£"
	case "EUR":
		return "€"
	default:
		return code + " "
	}
}
