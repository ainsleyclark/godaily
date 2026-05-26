// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGolangNuts_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/golang_nuts.xml")
	require.NoError(t, err)

	tt := map[string]struct {
		stub func() http.HandlerFunc
		want func(t *testing.T, items []news.Item, err error)
	}{
		"Bad Request": {
			stub: func() http.HandlerFunc {
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
		"OK": {
			stub: func() http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(fixture)
				}
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 2)
				assert.Equal(t, news.Item{
					Source:    news.SourceGolangNuts,
					Title:     "How to efficiently process large slices without allocation?",
					URL:       "http://www.mail-archive.com/golang-nuts@googlegroups.com/msg12345.html",
					Author:    &news.Author{Name: "Jane Developer"},
					Tag:       news.TagDiscussion,
					Score:     news.ScoreOf(news.SourceGolangNuts, news.TagDiscussion, 0, false),
					Published: time.Date(2026, time.May, 12, 8, 30, 0, 0, time.UTC),
				}, items[0])
			},
		},
		"Missing title prefix": {
			stub: func() http.HandlerFunc {
				const body = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>No prefix title</title>
      <link>https://www.mail-archive.com/golang-nuts@googlegroups.com/msg99999.html</link>
      <description>&lt;a href=&quot;...&quot;&gt;Someone&lt;/a&gt;</description>
      <pubDate>Thu, 01 Jan 2026 00:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(body))
				}
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, "No prefix title", items[0].Title)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub())
			defer s.Close()

			got, err := GolangNuts{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}
