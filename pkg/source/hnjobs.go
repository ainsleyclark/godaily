// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"encoding/json"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/source/ingest"
)

// HNJobs fetches Go-relevant comments from the latest "Ask HN: Who is hiring?"
// thread posted by the whoishiring account.
//
// Discovery is two-step because the thread ID changes every month:
//  1. Search Algolia for whoishiring stories and pick the most recent thread
//     whose title contains "who is hiring" (the same account also posts
//     "Who wants to be hired?" and "Freelancer?" threads).
//  2. Fetch the full comment tree via /items/<id> and filter top-level
//     children that mention Go or Golang as a whole word.
type HNJobs struct {
	storiesURL string
	itemURL    string // base for /items/<id>; the story ID is appended
	now        func() time.Time
}

var _ news.Fetcher = &HNJobs{}

func init() {
	news.Register(news.SourceHNJobs, func(cfg env.Config) news.Fetcher { return NewHNJobs(cfg) })
}

const (
	// search_by_date returns hits in descending creation order so the latest
	// month's "Who is hiring?" thread is always the first match. The plain
	// `search` endpoint orders by Algolia relevance and surfaces ancient
	// threads (e.g. "Who is hiring right now?" from 2020) first.
	hnJobsStoriesURL = "https://hn.algolia.com/api/v1/search_by_date?tags=story,author_whoishiring&hitsPerPage=10"
	hnJobsItemURL    = "https://hn.algolia.com/api/v1/items"
)

// NewHNJobs creates a Hacker News "Who is hiring?" client.
func NewHNJobs(_ env.Config) *HNJobs {
	return &HNJobs{
		storiesURL: hnJobsStoriesURL,
		itemURL:    hnJobsItemURL,
		now:        time.Now,
	}
}

// Fetch retrieves the latest Who-is-hiring thread and returns Go-relevant
// top-level comments as news items.
func (h HNJobs) Fetch(ctx context.Context) ([]news.Item, error) {
	stories, err := ingest.Fetch[hnJobsStoriesResponse](ctx, h.storiesURL, "hn whoishiring stories", json.Unmarshal)
	if err != nil {
		return nil, err
	}
	storyID := pickWhoIsHiringStory(stories.Hits)
	if storyID == "" {
		return nil, nil
	}

	thread, err := ingest.Fetch[hnThread](ctx, h.itemURL+"/"+storyID, "hn whoishiring thread", json.Unmarshal)
	if err != nil {
		return nil, err
	}

	now := h.now().UTC()
	comments := make([]hnJobsComment, 0, len(thread.Children))
	for _, child := range thread.Children {
		comments = append(comments, hnJobsComment{
			ID:        child.ID,
			Author:    child.Author,
			Text:      child.Text,
			CreatedAt: child.CreatedAt,
			ageDays:   hnJobsAgeDays(now, child.CreatedAt),
		})
	}
	return ingest.TransformAll(ctx, comments), nil
}

// pickWhoIsHiringStory returns the objectID of the most recent thread whose
// title contains "who is hiring". Hits from Algolia are already ordered by
// recency, so the first match is the latest month's thread.
func pickWhoIsHiringStory(hits []hnJobsStory) string {
	for _, h := range hits {
		if strings.Contains(strings.ToLower(h.Title), "who is hiring") {
			return h.ObjectID
		}
	}
	return ""
}

func hnJobsAgeDays(now, posted time.Time) int {
	if posted.IsZero() {
		return 0
	}
	days := int(now.Sub(posted).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func (c hnJobsComment) ShouldInclude() bool {
	if c.ID == 0 || c.Text == "" {
		return false
	}
	return hasGoWord(c.Text)
}

func (c hnJobsComment) EnrichmentURL() string { return "" }

func (c hnJobsComment) Transform() news.Item {
	title, snippet := parseHNJobComment(c.Text)
	company := parseHNJobCompany(title)
	if title == "" {
		title = company
	}

	goTitle := hasGoWord(title)
	hasSalaryInfo := hasSalary(c.Text)
	isRemoteRole := isRemote(c.Text)

	weight := news.SourceWeight(news.SourceHNJobs, news.TagJobs)
	score := weight * news.JobBoost(c.ageDays, goTitle, hasSalaryInfo, isRemoteRole)

	var author *news.Author
	if company != "" {
		author = &news.Author{Name: company}
	}

	return news.Item{
		Source:    news.SourceHNJobs,
		Title:     title,
		URL:       "https://news.ycombinator.com/item?id=" + strconv.FormatInt(c.ID, 10),
		Author:    author,
		Snippet:   snippet,
		Tag:       news.TagJobs,
		Published: c.CreatedAt,
		Score:     score,
	}
}

// maxHNJobTitleLen keeps the first-line job header from blowing out the email
// row. The whoishiring convention is "COMPANY | ROLE | LOC | REMOTE | $",
// which usually fits well under this.
const maxHNJobTitleLen = 140

var hnJobsTagRe = regexp.MustCompile(`<[^>]*>`)

// parseHNJobComment splits a Who-is-hiring comment into a one-line title and
// a body snippet. Three strategies, in order of preference:
//
//  1. <p>...</p> paragraph boundary — older HN comments preserve these.
//  2. The first \n inside the title-length window — Algolia's /items/
//     endpoint strips <p> tags but keeps embedded newlines.
//  3. Hard truncation at maxHNJobTitleLen on a word boundary. The whoishiring
//     convention is "COMPANY | ROLE | LOC | REMOTE | $" which usually fits;
//     when the header runs into the body without a separator (a frequent
//     Algolia quirk where "Full-time" jams against the next paragraph), the
//     hard cap still produces a scannable title and the rest surfaces in
//     the snippet.
func parseHNJobComment(text string) (title, snippet string) {
	if text == "" {
		return "", ""
	}

	if i := strings.Index(text, "</p>"); i > 0 {
		title = cleanTitle(text[:i])
		snippet = strings.TrimSpace(text[i+len("</p>"):])
		return truncateAtWord(title, maxHNJobTitleLen), snippet
	}

	cleaned := strings.TrimSpace(cleanTitle(text))
	if cleaned == "" {
		return "", ""
	}

	if i := strings.IndexByte(cleaned, '\n'); i > 0 && i <= maxHNJobTitleLen {
		return strings.TrimSpace(cleaned[:i]), strings.TrimSpace(cleaned[i:])
	}

	if len(cleaned) <= maxHNJobTitleLen {
		return cleaned, ""
	}
	cut := maxHNJobTitleLen - 3 // reserve room for the "..." marker
	if i := strings.LastIndexAny(cleaned[:cut], " \t"); i > cut/2 {
		cut = i
	}
	return strings.TrimRight(cleaned[:cut], " \t|·-") + "...", strings.TrimSpace(cleaned[cut:])
}

// truncateAtWord trims s to at most max bytes, cutting at the last whitespace
// before the limit so words aren't sliced. Reserves 3 bytes for the "..."
// marker. Returns s unchanged when it already fits.
func truncateAtWord(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max - 3
	if cut < 1 {
		return s[:max]
	}
	if i := strings.LastIndexAny(s[:cut], " \t"); i > cut/2 {
		cut = i
	}
	return strings.TrimRight(s[:cut], " \t|·-") + "..."
}

func cleanTitle(s string) string {
	s = hnJobsTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

// parseHNJobCompany returns the company name from the standard
// "COMPANY | ROLE | ..." header convention, or "" if the title doesn't
// follow the pattern.
func parseHNJobCompany(title string) string {
	if i := strings.Index(title, "|"); i > 0 {
		return strings.TrimSpace(title[:i])
	}
	return ""
}

type (
	hnJobsStoriesResponse struct {
		Hits []hnJobsStory `json:"hits"`
	}
	hnJobsStory struct {
		ObjectID string `json:"objectID"`
		Title    string `json:"title"`
	}
	hnThread struct {
		ID       int64           `json:"id"`
		Children []hnThreadChild `json:"children"`
	}
	hnThreadChild struct {
		ID        int64     `json:"id"`
		Author    string    `json:"author"`
		Text      string    `json:"text"`
		CreatedAt time.Time `json:"created_at"`
	}
	hnJobsComment struct {
		ID        int64
		Author    string
		Text      string
		CreatedAt time.Time
		ageDays   int
	}
)
