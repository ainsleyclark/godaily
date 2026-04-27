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

// redditOKResponse is a minimal r/golang listing with one external link post.
const redditOKResponse = `{
  "data": {
    "children": [
      {
        "data": {
          "title": "Go 1.23 released",
          "url": "https://go.dev/blog/go1.23",
          "author": "gopher",
          "selftext": "",
          "score": 500,
          "num_comments": 88,
          "created_utc": 1714000000.0,
          "permalink": "/r/golang/comments/abc123/go_123_released/"
        }
      }
    ]
  }
}`

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

func TestReddit_Fetch(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		stub http.HandlerFunc
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
				_, err := w.Write([]byte(redditOKResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceReddit,
					Title:     "Go 1.23 released",
					URL:       "https://go.dev/blog/go1.23",
					Author:    "gopher",
					Snippet:   "",
					Score:     500,
					Tag:       news.TagArticle,
					Comments:  88,
					Published: time.Unix(1714000000, 0).UTC(),
				}, items[0])
			},
		},
		"Self Post URL": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(redditSelfPostResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, "https://www.reddit.com/r/golang/comments/xyz789/ask_rgolang_best_go_books/", items[0].URL)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()
			got, err := Reddit{url: s.URL}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
