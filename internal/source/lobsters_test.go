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

const lobstersOKResponse = `[
  {
    "short_id": "abc123",
    "title": "Building a REST API in Go",
    "url": "https://example.com/go-rest-api",
    "score": 42,
    "comment_count": 7,
    "created_at": "2024-04-25T10:30:00.000-05:00",
    "description": "A tutorial on building REST APIs with Go.",
    "submitter_user": "gopher",
    "tags": ["go", "practices"]
  }
]`

const lobstersEmptyDescResponse = `[
  {
    "short_id": "def456",
    "title": "Go 1.24 is released",
    "url": "https://go.dev/blog/go1.24",
    "score": 100,
    "comment_count": 25,
    "created_at": "2024-04-26T12:00:00.000-05:00",
    "description": "",
    "submitter_user": "robpike",
    "tags": ["go"]
  }
]`

func TestLobsters_Fetch(t *testing.T) {
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
				_, err := w.Write([]byte(lobstersOKResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceLobsters,
					Title:     "Building a REST API in Go",
					URL:       "https://example.com/go-rest-api",
					Author:    "gopher",
					Snippet:   "A tutorial on building REST APIs with Go.",
					Score:     42,
					Tag:       news.TagArticle,
					Comments:  7,
					Published: time.Date(2024, 4, 25, 15, 30, 0, 0, time.UTC),
				}, items[0])
			},
		},
		"Empty Description": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(lobstersEmptyDescResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Empty(t, items[0].Snippet)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			s := httptest.NewServer(test.stub)
			defer s.Close()
			got, err := Lobsters{url: s.URL}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
