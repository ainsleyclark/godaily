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
	hnJobsStoriesURL = "https://hn.algolia.com/api/v1/search?tags=story,author_whoishiring&hitsPerPage=10"
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

// parseHNJobComment splits the first paragraph (the COMPANY | ROLE | ... line
// by convention) from the rest of the comment. The title is HTML-stripped and
// trimmed; the snippet is left as HTML and cleaned by ingest.sanitise.
func parseHNJobComment(text string) (title, snippet string) {
	if text == "" {
		return "", ""
	}
	end := strings.Index(text, "</p>")
	if end < 0 {
		return cleanTitle(text), ""
	}
	title = cleanTitle(text[:end])
	snippet = strings.TrimSpace(text[end+len("</p>"):])
	if len(title) > maxHNJobTitleLen {
		title = title[:maxHNJobTitleLen-3] + "..."
	}
	return title, snippet
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
