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

	"github.com/ainsleyclark/godaily/pkg/news"
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
					URL:       "https://groups.google.com/d/msgid/golang-nuts/CABcDeFgHiJkLmN%40mail.gmail.com",
					Author:    &news.Author{Name: "Jane Developer"},
					Snippet:   "I have been working on a hot path that processes millions of items per second. The current approach allocates a new slice for each batch. Has anyone found a good pattern using sync.Pool or similar?",
					Tag:       news.TagDiscussion,
					Score:     news.ScoreOf(news.SourceGolangNuts, news.TagDiscussion, 0, false),
					Published: time.Date(2026, time.May, 12, 8, 30, 0, 0, time.UTC),
				}, items[0])
			},
		},
		"Missing title prefix": {
			stub: func() http.HandlerFunc {
				const body = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>No prefix title</title>
    <link href="https://groups.google.com/d/msgid/golang-nuts/abc" rel="alternate"/>
    <author><name>Someone</name></author>
    <updated>2026-01-01T00:00:00Z</updated>
    <content type="html">Body text.</content>
  </entry>
</feed>`
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
