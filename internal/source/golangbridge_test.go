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

// golangBridgeEmptyResponse has an empty topics list.
const golangBridgeEmptyResponse = `{
  "topic_list": {
    "topics": []
  }
}`

func TestGolangBridge_Fetch(t *testing.T) {
	t.Parallel()

	// Real /latest.json response captured from forum.golangbridge.org.
	// GolangBridge has no enrichment hop (EnrichmentURL returns ""), so
	// topic URLs are constructed by the source from slug+id without any
	// network round-trip.
	fixture, err := os.ReadFile("testdata/golangbridge.json")
	require.NoError(t, err)

	tt := map[string]struct {
		stub http.HandlerFunc
		want func([]news.Item, error)
	}{
		"Bad Request": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			want: func(items []news.Item, err error) {
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write(fixture)
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 3)
				assert.Equal(t, news.Item{
					Source:    news.SourceGolangBridge,
					Title:     "An Unofficial Discourse User Reference Guide",
					URL:       "https://forum.golangbridge.org/t/an-unofficial-discourse-user-reference-guide/9738",
					Comments:  4,
					Tag:       news.TagArticle,
					Score:     1.0, // 13375 views saturates the curve; weight 1.0 * engagement 1.0
					Published: time.Date(2018, time.July, 3, 7, 31, 23, 214000000, time.UTC),
				}, items[0])
			},
		},
		"Empty Topics": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(golangBridgeEmptyResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Empty(t, items)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := GolangBridge{url: s.URL}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
