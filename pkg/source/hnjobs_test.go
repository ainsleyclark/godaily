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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPickWhoIsHiringStory(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		hits []hnJobsStory
		want string
	}{
		"Picks the hiring thread among siblings": {
			hits: []hnJobsStory{
				{ObjectID: "111", Title: "Ask HN: Who wants to be hired? (May 2026)"},
				{ObjectID: "222", Title: "Ask HN: Who is hiring? (May 2026)"},
				{ObjectID: "333", Title: "Ask HN: Freelancer? Seeking freelancer? (May 2026)"},
			},
			want: "222",
		},
		"Case-insensitive": {
			hits: []hnJobsStory{
				{ObjectID: "777", Title: "ASK HN: WHO IS HIRING? (JAN 2026)"},
			},
			want: "777",
		},
		"No match returns empty": {
			hits: []hnJobsStory{
				{ObjectID: "1", Title: "Show HN: Cool Go library"},
			},
			want: "",
		},
		"Empty hits returns empty": {hits: nil, want: ""},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, pickWhoIsHiringStory(test.hits))
		})
	}
}

func TestParseHNJobComment(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in          string
		wantTitle   string
		wantSnippet string
	}{
		"Standard header + body": {
			in:          `<p>Acme | Senior Go Engineer | Remote | $150k</p><p>We&#x27;re hiring an experienced Go developer to build distributed systems.</p>`,
			wantTitle:   "Acme | Senior Go Engineer | Remote | $150k",
			wantSnippet: `<p>We&#x27;re hiring an experienced Go developer to build distributed systems.</p>`,
		},
		"Header with anchor strips tags": {
			in:          `<p>Acme | <a href="https://acme.example">Backend Engineer</a> | Remote</p><p>Description here.</p>`,
			wantTitle:   "Acme | Backend Engineer | Remote",
			wantSnippet: `<p>Description here.</p>`,
		},
		"No paragraph boundary - whole text is title": {
			in:        `Single line job ad`,
			wantTitle: "Single line job ad",
		},
		"Empty input": {in: "", wantTitle: "", wantSnippet: ""},
		"Truncates over-long header": {
			// 200-char A-string forces truncation at maxHNJobTitleLen.
			in:        "<p>" + strings.Repeat("A", 200) + "</p><p>Body</p>",
			wantTitle: strings.Repeat("A", maxHNJobTitleLen-3) + "...",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			title, snippet := parseHNJobComment(test.in)
			assert.Equal(t, test.wantTitle, title)
			if test.wantSnippet != "" {
				assert.Equal(t, test.wantSnippet, snippet)
			}
		})
	}
}

func TestParseHNJobCompany(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"Standard pipe header":       {in: "Acme Corp | Senior Go Engineer | Remote", want: "Acme Corp"},
		"Trims whitespace":           {in: "  Big Tech   |  Role", want: "Big Tech"},
		"No pipe returns empty":      {in: "Single line", want: ""},
		"Leading pipe returns empty": {in: "| Role", want: ""},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, parseHNJobCompany(test.in))
		})
	}
}

func TestHNJobs_Fetch(t *testing.T) {
	t.Parallel()

	storyID := "42424242"

	// Two stories — the latest is the freelancer thread, ensuring
	// pickWhoIsHiringStory skips it and picks the hiring one.
	storiesBody := fmt.Sprintf(`{
		"hits":[
			{"objectID":"%s","title":"Ask HN: Who is hiring? (May 2026)"},
			{"objectID":"99","title":"Ask HN: Freelancer? Seeking freelancer?"}
		]
	}`, storyID)

	// Three top-level comments:
	//  - Acme: Go in title + salary + remote (passes filter, scores high)
	//  - PolyShop: "Go" in body only (passes filter, lower score)
	//  - Rustacean: no Go reference at all (filtered out)
	threadBody := `{
		"id":42424242,
		"children":[
			{
				"id":1001,
				"author":"acmehr",
				"created_at":"2026-05-02T10:00:00.000Z",
				"text":"<p>Acme | Senior Go Engineer | Remote | $150-180k</p><p>We&#x27;re hiring a Go developer for distributed systems work.</p>"
			},
			{
				"id":1002,
				"author":"polyshop",
				"created_at":"2026-05-02T11:00:00.000Z",
				"text":"<p>PolyShop | Backend Engineer | Onsite London</p><p>We use Python, Ruby, and some go services.</p>"
			},
			{
				"id":1003,
				"author":"rusty",
				"created_at":"2026-05-02T12:00:00.000Z",
				"text":"<p>RustCo | Rust Engineer | Remote | $120k</p><p>Pure Rust shop, no other languages.</p>"
			}
		]
	}`

	fixedNow := func() time.Time {
		return time.Date(2026, time.May, 3, 0, 0, 0, 0, time.UTC) // ~1 day after posts
	}

	tt := map[string]struct {
		stub func(storiesURL, itemURL string) http.HandlerFunc
		want func(t *testing.T, items []news.Item, err error)
	}{
		"Stories Bad Request": {
			stub: func(_, _ string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				}
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"No matching story returns empty": {
			stub: func(_, _ string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/items/") {
						t.Errorf("items endpoint should not be hit when no story matches: %s", r.URL.Path)
					}
					_, _ = w.Write([]byte(`{"hits":[{"objectID":"1","title":"Show HN: My Go project"}]}`))
				}
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.NoError(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(_, _ string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					switch {
					case strings.Contains(r.URL.Path, "/items/"):
						assert.Contains(t, r.URL.Path, storyID)
						_, _ = w.Write([]byte(threadBody))
					default:
						_, _ = w.Write([]byte(storiesBody))
					}
				}
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 2, "RustCo comment must be filtered out")

				// Order is preserved from the thread's children.
				acme := items[0]
				assert.Equal(t, news.SourceHNJobs, acme.Source)
				assert.Equal(t, news.TagJobs, acme.Tag)
				assert.Equal(t, "Acme | Senior Go Engineer | Remote | $150-180k", acme.Title)
				assert.Equal(t, "https://news.ycombinator.com/item?id=1001", acme.URL)
				require.NotNil(t, acme.Author)
				assert.Equal(t, "Acme", acme.Author.Name)
				assert.Contains(t, acme.Snippet, "distributed systems")
				assert.NotContains(t, acme.Snippet, "<p>", "snippet must be HTML-stripped by ingest")

				poly := items[1]
				assert.Equal(t, "PolyShop | Backend Engineer | Onsite London", poly.Title)
				// Acme scores higher: Go in title, salary, and remote vs. body-only Go reference.
				assert.Greater(t, acme.Score, poly.Score)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var storiesURL, itemURL string
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				test.stub(storiesURL, itemURL)(w, r)
			}))
			defer s.Close()
			storiesURL = s.URL + "/stories"
			itemURL = s.URL + "/items"

			got, err := HNJobs{
				storiesURL: storiesURL,
				itemURL:    itemURL,
				now:        fixedNow,
			}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestHNJobs_AgeDays(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 10, 0, 0, 0, 0, time.UTC)

	tt := map[string]struct {
		posted time.Time
		want   int
	}{
		"Same day":   {posted: now, want: 0},
		"Three days": {posted: now.Add(-72 * time.Hour), want: 3},
		"Future":     {posted: now.Add(24 * time.Hour), want: 0},
		"Zero":       {posted: time.Time{}, want: 0},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, hnJobsAgeDays(now, test.posted))
		})
	}
}
