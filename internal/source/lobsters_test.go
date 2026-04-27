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

// %s in each fixture is the test server URL — point story URLs at the same
// server so enrichment requests land on a non-HTML response and silently
// skip, keeping the test hermetic.
const lobstersOKResponseTpl = `[
  {
    "short_id": "abc123",
    "title": "Building a REST API in Go",
    "url": "%s",
    "comments_url": "https://lobste.rs/s/abc123/building_rest_api_go",
    "score": 42,
    "comment_count": 7,
    "created_at": "2024-04-25T10:30:00.000-05:00",
    "description": "A tutorial on building REST APIs with Go.",
    "submitter_user": "gopher",
    "tags": ["go", "practices"]
  }
]`

const lobstersEmptyDescResponseTpl = `[
  {
    "short_id": "def456",
    "title": "Go 1.24 is released",
    "url": "%s",
    "comments_url": "https://lobste.rs/s/def456/go_1_24_is_released",
    "score": 100,
    "comment_count": 25,
    "created_at": "2024-04-26T12:00:00.000-05:00",
    "description": "",
    "submitter_user": "robpike",
    "tags": ["go"]
  }
]`

// lobstersSelfPostResponse exercises the branch where the story URL points
// back at the discussion page (Lobsters self-post) — OriginalURL must stay
// empty in that case.
const lobstersSelfPostResponse = `[
  {
    "short_id": "ghi789",
    "title": "Ask Lobsters: favourite Go testing patterns?",
    "url": "https://lobste.rs/s/ghi789/ask_lobsters_favourite_go_testing",
    "comments_url": "https://lobste.rs/s/ghi789/ask_lobsters_favourite_go_testing",
    "score": 5,
    "comment_count": 3,
    "created_at": "2024-04-27T09:00:00.000-05:00",
    "description": "",
    "submitter_user": "asker",
    "tags": ["go", "ask"]
  }
]`

func TestLobsters_Fetch(t *testing.T) {
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
				body := fmt.Sprintf(lobstersOKResponseTpl, serverURL)
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
					Source:      news.SourceLobsters,
					Title:       "Building a REST API in Go",
					URL:         serverURL,
					OriginalURL: "https://lobste.rs/s/abc123/building_rest_api_go",
					Author:      "gopher",
					Snippet:     "A tutorial on building REST APIs with Go.",
					Tag:         news.TagArticle,
					Comments:    7,
					Score:       0.9566039969802683, // log(43)/log(51); weight 1.0 * engagement
					Published:   time.Date(2024, 4, 25, 15, 30, 0, 0, time.UTC),
				}, items[0])
			},
		},
		"Empty Description": {
			stub: func(serverURL string) http.HandlerFunc {
				body := fmt.Sprintf(lobstersEmptyDescResponseTpl, serverURL)
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(body))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Empty(t, items[0].Snippet)
			},
		},
		"Self Post": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(lobstersSelfPostResponse))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, "https://lobste.rs/s/ghi789/ask_lobsters_favourite_go_testing", items[0].URL)
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

			got, err := Lobsters{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err, s.URL)
		})
	}
}
