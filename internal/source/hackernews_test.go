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
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
)

// hnOKResponse is a minimal Algolia HN search response with one story hit.
// The story_text contains raw HTML tags and entities as returned by the API.
const hnOKResponse = `{
  "hits": [
    {
      "objectID": "43920000",
      "title": "Building a high-performance HTTP server in Go",
      "url": "https://example.com/go-http-server",
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
		stub http.HandlerFunc
		url  string
		want func([]news.Item, error)
	}{
		"Bad Request": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			want: func(items []news.Item, err error) {
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(hnOKResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceHN,
					Title:     "Building a high-performance HTTP server in Go",
					URL:       "https://example.com/go-http-server",
					Author:    "gopher42",
					Snippet:   "A deep dive into Go's net/http & stdlib.",
					Tag:       news.TagArticle,
					Comments:  42,
					Score:     1.2, // 350 points saturates the curve; weight 1.2 * engagement 1.0
					Published: time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC),
				}, items[0])
			},
		},
		"No Story URL": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(hnNoURLResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, "https://news.ycombinator.com/item?id=43920001", items[0].URL)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			url := s.URL
			if test.url != "" {
				url = test.url
			}

			got, err := HackerNews{url: url}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
