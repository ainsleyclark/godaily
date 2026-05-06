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
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoRelease_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/gorelease.json")
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
					_, err := w.Write(fixture)
					assert.NoError(t, err)
				}
			},
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				// Fixture contains 3 entries: 1 stable, 1 unstable RC, 1 stable.
				// ShouldInclude drops the RC, leaving 2 stable items.
				require.Len(t, items, 2)
				assert.Equal(t, news.Item{
					Source:  news.SourceGoRelease,
					Title:   "Go 1.26.2 released",
					URL:     "https://go.dev/doc/devel/release#go1.26.2",
					Snippet: "Stable Go release. See release notes for changes.",
					Tag:     news.TagRelease,
					Score:   1.0, // weight 2.0 * constantNoSignal 0.5
				}, items[0])
				assert.Equal(t, "Go 1.25.9 released", items[1].Title)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub())
			defer s.Close()

			got, err := GoRelease{url: s.URL, limit: 5}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestGoRelease_LimitTrims(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/gorelease.json")
	require.NoError(t, err)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer s.Close()

	// limit=1 keeps only the first entry from the upstream slice (which is stable),
	// so the unstable RC and the older stable release never reach Transform.
	got, err := GoRelease{url: s.URL, limit: 1}.Fetch(t.Context())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Go 1.26.2 released", got[0].Title)
}

func TestGoRelease_DateLookup(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/gorelease.json")
	require.NoError(t, err)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer s.Close()

	want := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)
	var gotURLs []string
	got, err := GoRelease{
		url:    s.URL,
		dlBase: "https://go.dev/dl/",
		limit:  5,
		dateFor: func(_ context.Context, fileURL string) time.Time {
			gotURLs = append(gotURLs, fileURL)
			return want
		},
	}.Fetch(t.Context())
	require.NoError(t, err)

	// Stable releases get the looked-up date; the RC is filtered out by ShouldInclude.
	require.Len(t, got, 2)
	assert.Equal(t, want, got[0].Published)
	assert.Equal(t, want, got[1].Published)
	// All three input releases (including the dropped RC) had their date queried.
	assert.Equal(t, []string{
		"https://go.dev/dl/go1.26.2.src.tar.gz",
		"https://go.dev/dl/go1.26rc3.src.tar.gz",
		"https://go.dev/dl/go1.25.9.src.tar.gz",
	}, gotURLs)
}
