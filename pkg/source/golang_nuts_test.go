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

func TestGolangNuts_Fetch(t *testing.T) {
	t.Parallel()

	// The included item's <link> is replaced with __SERVER_URL__ so snippet
	// enrichment requests land on the test server rather than the live
	// internet. The server returns the feed (text/xml) for the root path, so
	// enrichment finds no HTML meta description and leaves the snippet empty —
	// keeping the OK case deterministic.
	fixture, err := os.ReadFile("testdata/golang_nuts.xml")
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
					_, _ = w.Write([]byte(body))
				}
			},
			want: func(t *testing.T, items []news.Item, err error, serverURL string) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceGolangNuts,
					Title:     "How to efficiently process large slices without allocation?",
					URL:       serverURL,
					Author:    &news.Author{Name: "Jane Developer"},
					Tag:       news.TagDiscussion,
					Score:     news.ScoreOf(news.SourceGolangNuts, news.TagDiscussion, 0, false),
					Published: time.Date(2026, time.May, 12, 8, 30, 0, 0, time.UTC),
				}, items[0])
			},
		},
		"Enriches snippet from message page": {
			stub: func(serverURL string) http.HandlerFunc {
				feed := `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>[go-nuts] How to efficiently process large slices?</title>
      <link>` + serverURL + `/msg.html</link>
      <description>&lt;a href=&quot;...&quot;&gt;Jane Developer&lt;/a&gt;</description>
      <pubDate>Tue, 12 May 2026 08:30:00 GMT</pubDate>
    </item>
  </channel>
</rss>`
				return func(w http.ResponseWriter, r *http.Request) {
					if strings.HasSuffix(r.URL.Path, "/msg.html") {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(`<html><head>
<meta property="og:description" content="A discussion about reusing slice backing arrays to avoid per-call allocations.">
</head></html>`))
						return
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(feed))
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(
					t,
					"A discussion about reusing slice backing arrays to avoid per-call allocations.",
					items[0].Snippet,
				)
			},
		},
		"Filters reply threads": {
			stub: func(string) http.HandlerFunc {
				const body = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Re: [go-nuts] Some reply</title>
      <link>https://www.mail-archive.com/golang-nuts@googlegroups.com/msg00001.html</link>
      <description>&lt;a href=&quot;...&quot;&gt;Someone&lt;/a&gt;</description>
      <pubDate>Thu, 01 Jan 2026 00:00:00 GMT</pubDate>
    </item>
    <item>
      <title>[go-nuts] Re: Another reply</title>
      <link>https://www.mail-archive.com/golang-nuts@googlegroups.com/msg00002.html</link>
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
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, items)
			},
		},
		"Missing title prefix": {
			stub: func(serverURL string) http.HandlerFunc {
				body := `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>No prefix title</title>
      <link>` + serverURL + `</link>
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
			want: func(t *testing.T, items []news.Item, err error, _ string) {
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
			var serverURL string
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				test.stub(serverURL)(w, r)
			}))
			defer s.Close()
			serverURL = s.URL

			got, err := GolangNuts{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err, serverURL)
		})
	}
}
