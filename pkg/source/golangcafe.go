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

// GolangCafe scrapes the Golang.cafe job board. The site is a SvelteKit app
// behind Vercel bot protection, and its /rss feed 404s to proxies, but the
// listing page embeds the full set of listings as a SvelteKit hydration
// payload — a `jobPosts` array carrying explicit title/company/location/
// salary/remote/date fields. We parse that structured payload (richer and
// less brittle than the rendered markup); the JSON-LD on the page only lists
// bare job URLs, and the canonical /jobs/<slug> links live in the anchors,
// which we cross-reference by id. The board is Go-only, so every posting is
// relevant.
//
// ScraperAPI is used WITHOUT keep_headers so the proxy presents its own
// browser-like identity to clear Vercel's challenge; the data lives in the
// initial HTML, so no JS rendering is needed.
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
// (standard pool) when keys are available to clear Vercel's bot protection.
func NewGolangCafe(cfg env.Config) *GolangCafe {
	return &GolangCafe{
		url: ingest.ScraperURL(cfg.ScraperAPIKeys, golangCafeURL, ingest.WithoutPremium()),
		now: time.Now,
	}
}

// Fetch scrapes the listing page and returns its embedded job payload as items.
func (g GolangCafe) Fetch(ctx context.Context) ([]news.Item, error) {
	doc, err := ingest.FetchHTML(ctx, g.url, "golang cafe")
	if err != nil {
		return nil, err
	}

	raw := extractGolangCafeJobs(doc)
	urls := golangCafeJobURLs(doc)
	now := g.now().UTC()

	jobs := make([]golangCafeJob, 0, len(raw))
	for _, r := range raw {
		jobs = append(jobs, newGolangCafeJob(r, urls, now))
	}
	return ingest.TransformAll(ctx, jobs), nil
}

// golangCafeRawJob mirrors the fields we need from a SvelteKit `jobPosts`
// entry. Unknown fields (imageId, tags, websites, image, …) are ignored.
type golangCafeRawJob struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Company    string `json:"company"`
	Location   string `json:"location"`
	Link       string `json:"link"`
	Currency   string `json:"currency"`
	Remote     string `json:"remote"`
	SalaryFrom string `json:"salaryFrom"`
	SalaryTo   string `json:"salaryTo"`
	Date       int64  `json:"date"`
}

type golangCafeJob struct {
	title     string
	company   string
	location  string
	url       string
	salary    string
	remote    bool
	ageDays   int
	published time.Time
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
		Snippet:   j.salary,
		Tag:       news.TagJobs,
		Score:     score,
		Published: j.published,
	}
}

// newGolangCafeJob projects a raw payload entry onto our intermediate type,
// resolving the canonical /jobs/<slug> URL by id and stamping the real posting
// date so the digest's recency window can age out stale listings.
func newGolangCafeJob(r golangCafeRawJob, urls map[string]string, now time.Time) golangCafeJob {
	url := urls[r.ID]
	if url == "" && strings.HasPrefix(r.Link, "http") {
		url = r.Link // fall back to the application URL when no canonical link
	}

	published := now
	if r.Date > 0 {
		published = time.UnixMilli(r.Date).UTC()
	}
	ageDays := int(now.Sub(published).Hours() / 24)
	if ageDays < 0 {
		ageDays = 0
	}

	return golangCafeJob{
		title:     strings.TrimSpace(r.Title),
		company:   strings.TrimSpace(r.Company),
		location:  strings.TrimSpace(r.Location),
		url:       url,
		salary:    golangCafeSalary(r.Currency, r.SalaryFrom, r.SalaryTo),
		remote:    r.Remote != "" && r.Remote != "on_site",
		ageDays:   ageDays,
		published: published,
	}
}

// golangCafeSalary renders the payload's currency symbol and salary bounds as
// e.g. "£90k–£120k", a single-bound variant, or "" when undisclosed. The
// payload's currency is already a symbol ("$", "£", "€").
func golangCafeSalary(currency, from, to string) string {
	f := digitsToInt(from)
	t := digitsToInt(to)
	if f == 0 && t == 0 {
		return ""
	}
	sym := strings.TrimSpace(currency)
	if sym == "" {
		sym = "$"
	}
	switch {
	case f > 0 && t > 0:
		return sym + formatThousands(float64(f)) + "–" + sym + formatThousands(float64(t))
	case f > 0:
		return sym + formatThousands(float64(f)) + "+"
	default:
		return "up to " + sym + formatThousands(float64(t))
	}
}

// digitsToInt parses an integer from a string that may carry spaces or other
// separators (the payload formats salaries as e.g. "200 000").
func digitsToInt(s string) int {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return 0
	}
	n, _ := strconv.Atoi(b.String())
	return n
}

// golangCafeJobURLs maps a job id to its canonical /jobs/<slug> URL by reading
// the listing anchors, whose slug always ends in the id.
func golangCafeJobURLs(doc *goquery.Document) map[string]string {
	out := map[string]string{}
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if !strings.HasPrefix(href, "/jobs/") {
			return
		}
		slug := strings.TrimPrefix(href, "/jobs/")
		if i := strings.LastIndex(slug, "-"); i >= 0 && i+1 < len(slug) {
			out[slug[i+1:]] = golangCafeURL + href
		}
	})
	return out
}

// extractGolangCafeJobs finds the SvelteKit hydration script and decodes its
// `jobPosts` array. The payload is a JS object literal (unquoted keys), so we
// isolate the array and quote its keys before unmarshalling.
func extractGolangCafeJobs(doc *goquery.Document) []golangCafeRawJob {
	var jobs []golangCafeRawJob
	doc.Find("script").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		txt := s.Text()
		idx := strings.Index(txt, "jobPosts:")
		if idx < 0 {
			return true
		}
		rel := strings.IndexByte(txt[idx:], '[')
		if rel < 0 {
			return true
		}
		arr, ok := extractJSArray(txt[idx+rel:])
		if !ok {
			return true
		}
		if err := json.Unmarshal([]byte(jsObjectToJSON(arr)), &jobs); err != nil {
			return true // keep scanning; another script might carry it
		}
		return false
	})
	return jobs
}

// extractJSArray returns the balanced "[ … ]" beginning at s[0], tracking string
// literals so brackets inside descriptions don't terminate it early.
func extractJSArray(s string) (string, bool) {
	if len(s) == 0 || s[0] != '[' {
		return "", false
	}
	depth, inStr, esc := 0, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '[', '{':
			depth++
		case ']', '}':
			depth--
			if depth == 0 {
				return s[:i+1], true
			}
		}
	}
	return "", false
}

// jsObjectToJSON quotes the unquoted identifier keys in a JS object literal so
// it parses as JSON. It is string-aware, so identifiers inside string values
// (e.g. "Salary:" in a description) are left untouched, and bareword values
// like true/false/null pass through unquoted.
func jsObjectToJSON(s string) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/16)
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			b.WriteByte(c)
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			b.WriteByte(c)
			continue
		}
		if isIdentStart(c) {
			j := i
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			ident := s[i:j]
			k := j
			for k < len(s) && (s[k] == ' ' || s[k] == '\t' || s[k] == '\n' || s[k] == '\r') {
				k++
			}
			if k < len(s) && s[k] == ':' {
				b.WriteByte('"')
				b.WriteString(ident)
				b.WriteByte('"')
			} else {
				b.WriteString(ident)
			}
			i = j - 1
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
