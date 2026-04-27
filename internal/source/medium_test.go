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
	"testing"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
)

const mediumOKResponse = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel>
    <item>
      <title>Understanding Go Interfaces</title>
      <link>https://medium.com/@gopher/understanding-go-interfaces</link>
      <dc:creator>Gopher Dev</dc:creator>
      <description><![CDATA[<p>Interfaces in Go are <strong>implicit</strong>.</p>]]></description>
      <pubDate>Thu, 25 Apr 2024 10:00:00 +0000</pubDate>
      <category>golang</category>
      <category>programming</category>
    </item>
  </channel>
</rss>`

func TestMedium_Fetch(t *testing.T) {
	t.Parallel()

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
				_, err := w.Write([]byte(mediumOKResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceMedium,
					Title:     "Understanding Go Interfaces",
					URL:       "https://medium.com/@gopher/understanding-go-interfaces",
					Author:    "Gopher Dev",
					Snippet:   "Interfaces in Go are implicit .",
					Tag:       news.TagArticle,
					Score:     0.25, // no signal: weight 0.5 * constant 0.5
					Published: time.Date(2024, 4, 25, 10, 0, 0, 0, time.UTC),
				}, items[0])
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()
			got, err := Medium{url: s.URL}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
