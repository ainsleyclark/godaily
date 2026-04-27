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

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYouTube_Fetch(t *testing.T) {
	t.Parallel()

	// YouTube Data API v3 sample response shape (matches Google's documented
	// schema). YouTube has no enrichment hop (EnrichmentURL returns ""),
	// so no URL substitution is required.
	fixture, err := os.ReadFile("testdata/youtube.json")
	require.NoError(t, err)

	tt := map[string]struct {
		key  string
		stub http.HandlerFunc
		want func([]news.Item, error)
	}{
		"Missing API Key": {
			key:  "",
			stub: nil,
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Nil(t, items)
			},
		},
		"Bad Request": {
			key: "test-key",
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			want: func(items []news.Item, err error) {
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			key: "test-key",
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write(fixture)
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceYouTube,
					Title:     "Go Concurrency Patterns",
					URL:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
					ImageURL:  "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
					Author:    "GopherCon",
					Snippet:   "An introduction to concurrency patterns in Go.",
					Tag:       news.TagVideo,
					Score:     0.5, // no signal: weight 1.0 * constant 0.5
					Published: time.Date(2024, 4, 25, 14, 0, 0, 0, time.UTC),
				}, items[0])
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			url := "http://unused"
			if test.stub != nil {
				s := httptest.NewServer(test.stub)
				defer s.Close()
				url = s.URL
			}
			got, err := YouTube{url: url, key: test.key}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
