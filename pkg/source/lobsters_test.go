// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// Real /t/go.json response captured from lobste.rs — every story's "url"
	// field is replaced with __SERVER_URL__ so enrichment requests land on
	// the test server (which returns JSON, not HTML, and silently skips).
	fixture, err := os.ReadFile("testdata/lobsters.json")
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
			want: func(t *testing.T, items []news.Item, err error, serverURL string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 3)
				assert.Equal(t, news.Item{
					Source:      news.SourceLobsters,
					Title:       "Swissing a table",
					URL:         serverURL,
					OriginalURL: "https://lobste.rs/s/2lzsw6/swissing_table",
					Author:      &news.Author{Username: "carlana", ProfileURL: "https://lobste.rs/u/carlana"},
					Tag:         news.TagDiscussion,
					Comments:    4,
					Score:       0.9042486604437607, // log(35)/log(51); weight 1.0 * engagement
					Published:   time.Date(2026, 4, 26, 14, 19, 13, 0, time.UTC),
				}, items[0])
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
