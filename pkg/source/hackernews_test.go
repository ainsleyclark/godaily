// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHnWindow(t *testing.T) {
	t.Parallel()

	// Monday 2026-05-11 01:00 UTC
	monday := time.Date(2026, time.May, 11, 1, 0, 0, 0, time.UTC)
	// Tuesday 2026-05-12 01:00 UTC
	tuesday := time.Date(2026, time.May, 12, 1, 0, 0, 0, time.UTC)

	t.Run("Monday returns Saturday+Sunday window", func(t *testing.T) {
		t.Parallel()
		start, end := hnWindow(monday)
		assert.Equal(t, time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC), start) // Saturday
		assert.Equal(t, time.Date(2026, time.May, 11, 0, 0, 0, 0, time.UTC), end)  // Monday midnight
	})

	t.Run("Non-Monday returns yesterday window", func(t *testing.T) {
		t.Parallel()
		start, end := hnWindow(tuesday)
		assert.Equal(t, time.Date(2026, time.May, 11, 0, 0, 0, 0, time.UTC), start) // Monday midnight
		assert.Equal(t, time.Date(2026, time.May, 12, 0, 0, 0, 0, time.UTC), end)   // Tuesday midnight
	})
}

func TestHnURL(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.May, 11, 0, 0, 0, 0, time.UTC)
	raw := hnURL(start, end)

	u, err := url.Parse(raw)
	require.NoError(t, err)

	// numericFilters value must be present and percent-encoded (no raw >, <, or ,).
	filter := u.Query().Get("numericFilters")
	assert.NotEmpty(t, filter)
	assert.NotContains(t, u.RawQuery, ">", "raw > must be percent-encoded in query string")
	assert.NotContains(t, u.RawQuery, "<", "raw < must be percent-encoded in query string")
	assert.Contains(t, u.RawQuery, "%3E")
	assert.Contains(t, u.RawQuery, "%3C")
}

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

	// Real Algolia response captured from hn.algolia.com — every hit's "url"
	// field is replaced with __SERVER_URL__ so enrichment requests land on
	// the test server (which returns JSON, not HTML, and silently skips).
	fixture, err := os.ReadFile("testdata/hackernews.json")
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
				assert.Len(t, items, 2)
				assert.Equal(t, news.Item{
					Source:      news.SourceHN,
					Title:       "I learned Rust with rustlings, so I built the same thing for Go",
					URL:         serverURL,
					OriginalURL: "https://news.ycombinator.com/item?id=47912690",
					Author:      &news.Author{Username: "ichihiroy", ProfileURL: "https://news.ycombinator.com/user?id=ichihiroy"},
					Tag:         news.TagDiscussion,
					Comments:    0,
					Score:       0.4230994425333170, // 3 points: log(4)/log(101); weight 1.2
					Published:   time.Date(2026, time.April, 26, 18, 40, 36, 0, time.UTC),
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
