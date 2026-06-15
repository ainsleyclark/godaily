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

	// __SERVER_URL__ in the fixture is rewritten to the test server so body
	// enrichment doesn't hit the live internet.
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
		"Enriches snippet from MHonArc message body": {
			stub: func(serverURL string) http.HandlerFunc {
				feed := `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>[go-nuts] FullStack Go framework</title>
      <link>` + serverURL + `/msg.html</link>
      <description>&lt;a href=&quot;...&quot;&gt;Jane Developer&lt;/a&gt;</description>
      <pubDate>Tue, 12 May 2026 08:30:00 GMT</pubDate>
    </item>
  </channel>
</rss>`
				page := `<html><head><title>[go-nuts] FullStack Go framework</title></head><body>
<!--X-Body-of-Message-->
<pre>&gt; someone wrote a quoted reply line
Some things force us to evolve. I built my own *Go* FullStack framework.
</pre>
<!--X-Body-of-Message-End-->
</body></html>`
				return func(w http.ResponseWriter, r *http.Request) {
					if strings.HasSuffix(r.URL.Path, "/msg.html") {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(page))
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
					"Some things force us to evolve. I built my own Go FullStack framework.",
					items[0].Snippet,
				)
			},
		},
		"Strips signature and Google Groups footer from snippet": {
			stub: func(serverURL string) http.HandlerFunc {
				feed := `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>[go-nuts] pkg.go.dev API</title>
      <link>` + serverURL + `/msg.html</link>
      <description>&lt;a href=&quot;...&quot;&gt;Jane Developer&lt;/a&gt;</description>
      <pubDate>Tue, 12 May 2026 08:30:00 GMT</pubDate>
    </item>
  </channel>
</rss>`
				page := `<html><head><title>[go-nuts] pkg.go.dev API</title></head><body>
<!--X-Body-of-Message-->
<pre>Thanks Go team for exposing the pkg.go.dev API.
--
You received this message because you are subscribed to the Google Groups
&quot;golang-nuts&quot; group.
To unsubscribe from this group and stop receiving emails from it, send an email
to golang-nuts+unsubscribe@googlegroups.com.
</pre>
<!--X-Body-of-Message-End-->
</body></html>`
				return func(w http.ResponseWriter, r *http.Request) {
					if strings.HasSuffix(r.URL.Path, "/msg.html") {
						w.Header().Set("Content-Type", "text/html")
						_, _ = w.Write([]byte(page))
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
				assert.Equal(t, "Thanks Go team for exposing the pkg.go.dev API.", items[0].Snippet)
				assert.NotContains(t, items[0].Snippet, "You received this message")
				assert.NotContains(t, items[0].Snippet, "unsubscribe")
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

func TestExtractMHonArcBody(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"Extracts body between markers": {
			in:   "<html><!--X-Body-of-Message--><pre>Hello there</pre><!--X-Body-of-Message-End--></html>",
			want: "<pre>Hello there</pre>",
		},
		"Drops quoted reply lines": {
			in:   "<!--X-Body-of-Message-->\n&gt; old quoted line\nmy own words\n<!--X-Body-of-Message-End-->",
			want: "\nmy own words\n",
		},
		"No start marker": {
			in:   "<html><pre>Hello</pre></html>",
			want: "",
		},
		"No end marker or fallback": {
			in:   "<!--X-Body-of-Message--><pre>Hello</pre>",
			want: "",
		},
		"Falls back to msgButtons div when end marker absent": {
			in:   `<!--X-Body-of-Message--><pre>Hi there</pre></div><div class="msgButtons ">…`,
			want: `<pre>Hi there</pre></div>`,
		},
		"Cuts signature and Google Groups footer at the delimiter": {
			in: "<!--X-Body-of-Message--><pre>My actual message.\n" +
				"-- \n" +
				"You received this message because you are subscribed to the Google Groups\n" +
				"\"golang-nuts\" group.\n" +
				"To unsubscribe from this group...\n" +
				"</pre><!--X-Body-of-Message-End-->",
			want: "<pre>My actual message.\n",
		},
		"Keeps body when no signature delimiter present": {
			in:   "<!--X-Body-of-Message--><pre>Just the body -- no delimiter here.</pre><!--X-Body-of-Message-End-->",
			want: "<pre>Just the body -- no delimiter here.</pre>",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, extractMHonArcBody(test.in))
		})
	}
}

// Locks the extractor to real mail-archive.com markup, not just synthetic fixtures.
func TestExtractMHonArcBody_RealPage(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/golang_nuts_msg.html")
	require.NoError(t, err)

	body := extractMHonArcBody(string(raw))
	require.NotEmpty(t, body, "extractor returned empty body for real page")
	assert.Contains(t, body, "Some things force us to evolve")
	assert.Contains(t, body, "FullStack")
	assert.NotContains(t, body, "msgButtons")
	assert.NotContains(t, body, "View by thread")
	// The signature delimiter and the Google Groups footer that follows it
	// must be cut from the extracted body.
	assert.NotContains(t, body, "You received this message because")
	assert.NotContains(t, body, "To unsubscribe from this group")
}
