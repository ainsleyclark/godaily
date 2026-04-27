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
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
)

// %s is filled with the test server URL — the story URL points at the same
// server so enrichment requests get a non-HTML response and silently skip.
const hnOKResponseTpl = `{
  "hits": [
    {
      "objectID": "43920000",
      "title": "Building a high-performance HTTP server in Go",
      "url": "%s",
      "author": "gopher42",
      "story_text": "<p>A deep dive into Go&#x27;s net/http &amp; stdlib.",
      "points": 350,
      "num_comments": 42,
      "created_at": "2026-04-20T10:00:00.000Z"
    }
  ]
}`

// hnNoURLResponse is a hit where the url field is absent (Ask HN / self-post),
// exercising the HN permalink fallback in transform().
const hnNoURLResponse = `{
  "hits": [
    {
      "objectID": "43920001",
      "title": "Ask HN: Best resources for learning Go in 2026?",
      "url": "",
      "author": "curious_dev",
      "story_text": "Looking for up-to-date learning resources.",
      "points": 120,
      "num_comments": 30,
      "created_at": "2026-04-21T08:30:00.000Z"
    }
  ]
}`

func TestHackerNews_Fetch(t *testing.T) {
	t.Parallel()

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
				body := fmt.Sprintf(hnOKResponseTpl, serverURL)
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(body))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, serverURL string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:      news.SourceHN,
					Title:       "Building a high-performance HTTP server in Go",
					URL:         serverURL,
					OriginalURL: "https://news.ycombinator.com/item?id=43920000",
					Author:      "gopher42",
					Snippet:     "A deep dive into Go's net/http & stdlib.",
					Tag:         news.TagArticle,
					Comments:    42,
					Score:       1.2, // 350 points saturates the curve; weight 1.2 * engagement 1.0
					Published:   time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC),
				}, items[0])
			},
		},
		"No Story URL": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(hnNoURLResponse))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, "https://news.ycombinator.com/item?id=43920001", items[0].URL)
				assert.Empty(t, items[0].OriginalURL)
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

			got, err := HackerNews{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err, s.URL)
		})
	}
}
