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
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Card with no <h2><a> — parser must skip it without aborting.
	trendingMalformedCard = `<html><body>
<article class="Box-row"><p>no link here</p></article>
<article class="Box-row">
  <h2 class="h3 lh-condensed"><a href="/owner/repo">owner / repo</a></h2>
  <p>desc</p>
  <span class="d-inline-block float-sm-right">42 stars today</span>
</article>
</body></html>`

	// Valid card with no stars-today span — score should fall to the default floor.
	trendingNoStarsCard = `<html><body>
<article class="Box-row">
  <h2 class="h3 lh-condensed"><a href="/owner/repo">owner / repo</a></h2>
  <p>desc</p>
</article>
</body></html>`
)

func TestGitHubTrending_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/github_trending.html")
	require.NoError(t, err)

	const trendingPath = "/trending/go"

	// expectedPublished mirrors the production formula so the assertion remains
	// stable regardless of when the test runs.
	expectedPublished := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(12 * time.Hour)

	tt := map[string]struct {
		stub func(serverURL string) http.HandlerFunc
		want func(t *testing.T, items []news.Item, err error, serverURL string)
	}{
		"Bad Request": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != trendingPath {
						http.NotFound(w, r) // enrichment GETs land here and silently skip
						return
					}
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusOK)
					_, err := w.Write(fixture)
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, serverURL string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 18)

				first := items[0]
				assert.Equal(t, news.SourceGitHubTrending, first.Source)
				assert.Equal(t, "gastownhall/beads", first.Title)
				assert.Equal(t, &news.Author{Username: "gastownhall", ProfileURL: "https://github.com/gastownhall"}, first.Author)
				assert.Equal(t, serverURL+"/gastownhall/beads", first.URL)
				assert.Equal(t, "Beads - A memory upgrade for your coding agent", first.Snippet)
				assert.Equal(t, news.TagTrending, first.Tag)
				assert.Equal(t, expectedPublished, first.Published)
				// 485 stars saturates the curve (sat=200) → engagement clamps to
				// 1.0; default weight 1.0 → score 1.0.
				assert.InDelta(t, 1.0, first.Score, 0.001)
			},
		},
		"Malformed Card Skipped": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != trendingPath {
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					_, err := w.Write([]byte(trendingMalformedCard))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, serverURL string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, serverURL+"/owner/repo", items[0].URL)
			},
		},
		"Missing Stars Today": {
			stub: func(string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != trendingPath {
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					_, err := w.Write([]byte(trendingNoStarsCard))
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				// Zero signal lands on the default engagement floor (0.1) × weight (1.0).
				assert.InDelta(t, 0.1, items[0].Score, 0.001)
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

			got, err := GitHubTrending{url: s.URL + trendingPath}.Fetch(t.Context())
			test.want(t, got, err, s.URL)
		})
	}
}
