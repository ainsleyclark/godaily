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
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// redditSelfPostResponse is a self-post whose URL points back to Reddit,
// exercising the permalink fallback in transform().
const redditSelfPostResponse = `{
  "data": {
    "children": [
      {
        "data": {
          "title": "Ask r/golang: best Go books?",
          "url": "https://www.reddit.com/r/golang/comments/xyz789/ask_rgolang_best_go_books/",
          "author": "learner",
          "selftext": "Looking for recommendations.",
          "score": 42,
          "num_comments": 15,
          "created_utc": 1714100000.0,
          "permalink": "/r/golang/comments/xyz789/ask_rgolang_best_go_books/"
        }
      }
    ]
  }
}`

func TestRedditChild_ShouldInclude(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input redditChild
		want  bool
	}{
		"Included": {
			input: redditChild{Data: redditPost{Title: "New concurrency patterns in Go"}},
			want:  true,
		},
		"Help in title": {
			input: redditChild{Data: redditPost{Title: "Need help with goroutines"}},
			want:  false,
		},
		"Feedback in title": {
			input: redditChild{Data: redditPost{Title: "Feedback on my Go project"}},
			want:  false,
		},
		"Feedback in body": {
			input: redditChild{Data: redditPost{Title: "My new library", SelfText: "Please give me feedback on this."}},
			want:  false,
		},
		"Feedback case insensitive title": {
			input: redditChild{Data: redditPost{Title: "FEEDBACK wanted on my API design"}},
			want:  false,
		},
		"Feedback case insensitive body": {
			input: redditChild{Data: redditPost{Title: "Go microservices", SelfText: "Looking for FEEDBACK on the architecture."}},
			want:  false,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := test.input.ShouldInclude()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestReddit_Fetch(t *testing.T) {
	t.Parallel()

	// Real /r/golang/new.json response captured from reddit.com — every
	// child's external "url" field is replaced with __SERVER_URL__ so
	// enrichment lands on the test server (self-post URLs that point back
	// at reddit.com are kept verbatim and skip enrichment via the source).
	fixture, err := os.ReadFile("testdata/reddit.json")
	require.NoError(t, err)

	tt := map[string]struct {
		stub func(serverURL string) http.HandlerFunc
		want func(t *testing.T, items []news.Item, err error, serverURL string)
	}{
		"Bad Request": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(serverURL string) http.HandlerFunc {
				body := strings.ReplaceAll(string(fixture), "__SERVER_URL__", serverURL)
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(body))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 3)
				assert.Equal(t, news.Item{
					Source:    news.SourceReddit,
					Title:     "Small Projects",
					URL:       "https://www.reddit.com/r/golang/comments/1sxd6ei/small_projects/",
					Author:    &news.Author{Username: "AutoModerator", ProfileURL: "https://www.reddit.com/user/AutoModerator"},
					Snippet:   "This is the weekly thread for Small Projects. The point of this thread is to have looser posting standards than the main board. As such, projects are pretty much only removed from here by the mods for",
					Tag:       news.TagDiscussion,
					Comments:  0,
					Score:     0.23804628387473528, // 2 score: log(3)/log(101); weight 1.0
					Published: time.Date(2026, 4, 27, 19, 0, 54, 0, time.UTC),
				}, items[0])
			},
		},
		"Self Post URL": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(redditSelfPostResponse))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, "https://www.reddit.com/r/golang/comments/xyz789/ask_rgolang_best_go_books/", items[0].URL)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var serverURL string
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				test.stub(serverURL)(w, r)
			}))
			defer s.Close()
			serverURL = s.URL

			got, err := Reddit{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err, s.URL)
		})
	}
}
