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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// atomNoLinkResponse is a feed where the entry's link has a non-alternate rel,
// exercising the url() fallback that returns an empty string.
const atomNoLinkResponse = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>No Link Entry</title>
    <link rel="self" href="https://go.dev/blog/feed.atom"/>
    <author><name>The Go Team</name></author>
    <published>2024-08-13T00:00:00Z</published>
    <summary>Entry with no alternate link.</summary>
  </entry>
</feed>`

func TestGoBlog_Fetch(t *testing.T) {
	t.Parallel()

	// Real Atom feed captured from go.dev/blog/feed.atom — content bodies
	// stripped to keep the fixture lean, and each entry's alternate link
	// replaced with __SERVER_URL__ so enrichment lands on the test server.
	fixture, err := os.ReadFile("testdata/godevblog.atom")
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
					Source:    news.SourceGoBlog,
					Title:     "Type Construction and Cycle Detection",
					URL:       serverURL,
					Author:    &news.Author{Name: "Mark Freeman"},
					Snippet:   "Go 1.26 simplifies type construction and enhances cycle detection for certain kinds of recursive types.",
					Tag:       news.TagArticle,
					Score:     1.0, // no signal but official: weight 2.0 * constant 0.5
					Published: time.Date(2026, time.March, 24, 0, 0, 0, 0, time.UTC),
				}, items[0])
			},
		},
		"No Alternate Link": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(atomNoLinkResponse))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, "", items[0].URL)
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

			got, err := GoBlog{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err, s.URL)
		})
	}
}
