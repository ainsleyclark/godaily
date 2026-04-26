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
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// atomOKResponse is a minimal Atom 1.0 feed with one entry used in the OK test case.
const atomOKResponse = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>The Go Blog</title>
  <entry>
    <title>Go 1.23 Released</title>
    <link rel="alternate" href="https://go.dev/blog/go1.23"/>
    <author><name>The Go Team</name></author>
    <published>2024-08-13T00:00:00Z</published>
    <summary>Go 1.23 is now available.</summary>
  </entry>
</feed>`

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

	tt := map[string]struct {
		stub http.HandlerFunc
		url  string
		want func([]news.Item, error)
	}{
		"Error Creating Request": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			url: ":@!£$",
			want: func(_ []news.Item, err error) {
				assert.Error(t, err)
			},
		},
		"Bad Request": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			want: func(items []news.Item, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "unexpected status code")
				assert.Nil(t, items)
			},
		},
		"Decode Error": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`not xml at all <<>>`))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "parsing response")
				assert.Nil(t, items)
			},
		},
		"OK": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(atomOKResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceGoBlog,
					Title:     "Go 1.23 Released",
					URL:       "https://go.dev/blog/go1.23",
					Author:    "The Go Team",
					Snippet:   "Go 1.23 is now available.",
					Tag:       news.TagArticle,
					Published: time.Date(2024, time.August, 13, 0, 0, 0, 0, time.UTC),
				}, items[0])
			},
		},
		"No Alternate Link": {
			stub: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(atomNoLinkResponse))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, "", items[0].URL)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			s := httptest.NewServer(test.stub)
			defer s.Close()

			url := s.URL
			if test.url != "" {
				url = test.url
			}

			c := GoBlog{
				http: s.Client(),
				url:  url,
			}

			got, err := c.Fetch(t.Context())
			test.want(got, err)
		})
	}

	t.Run("Do Error", func(t *testing.T) {
		f := NewGoBlog()
		f.http = &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			}),
		}

		_, err := f.Fetch(t.Context())
		assert.ErrorContains(t, err, "fetch go blog")
	})
}
