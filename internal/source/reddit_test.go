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

// %s is filled with the test server URL — the post URL points at the same
// server so enrichment requests get a non-HTML response and silently skip.
const redditOKResponseTpl = `{
  "data": {
    "children": [
      {
        "data": {
          "title": "Go 1.23 released",
          "url": "%s",
          "author": "gopher",
          "selftext": "",
          "score": 500,
          "num_comments": 88,
          "created_utc": 1714000000.0,
          "permalink": "/r/golang/comments/abc123/go_123_released/",
          "preview": {
            "images": [
              {"source": {"url": "https://preview.redd.it/abc.jpg?width=640&amp;auto=webp"}}
            ]
          },
          "thumbnail": "https://b.thumbs.redditmedia.com/x.jpg"
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
				body := fmt.Sprintf(redditOKResponseTpl, serverURL)
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
					Source:    news.SourceReddit,
					Title:     "Go 1.23 released",
					URL:       serverURL,
					ImageURL:  "https://preview.redd.it/abc.jpg?width=640&auto=webp",
					Author:    "gopher",
					Snippet:   "",
					Tag:       news.TagArticle,
					Comments:  88,
					Score:     1.0, // score 500 saturates the curve; weight 1.0 * 1.0
					Published: time.Unix(1714000000, 0).UTC(),
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
