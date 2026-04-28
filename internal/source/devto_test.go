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

func TestDevTo_Fetch(t *testing.T) {
	t.Parallel()

	// Real /api/articles?tag=go response captured from dev.to. DevTo has no
	// enrichment hop (EnrichmentURL returns ""), so item URLs stay verbatim.
	fixture, err := os.ReadFile("testdata/devto.json")
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
					Source:   news.SourceDevTo,
					Title:    "🚀 Building a CRUD API in Go with PostgreSQL (Step-by-Step)",
					URL:      "https://dev.to/ahmedraza_fyntune/building-a-crud-api-in-go-with-postgresql-step-by-step-2n34",
					ImageURL: "https://media2.dev.to/dynamic/image/width=1000,height=420,fit=cover,gravity=auto,format=auto/https%3A%2F%2Fdev-to-uploads.s3.amazonaws.com%2Fuploads%2Farticles%2Fas19p9v7rfz1vj7ip25e.png",
					Author: &news.Author{
						Name:       "Ahmed Raza Idrisi",
						Username:   "ahmedraza_fyntune",
						AvatarURL:  "https://media2.dev.to/dynamic/image/width=640,height=640,fit=cover,gravity=auto,format=auto/https%3A%2F%2Fdev-to-uploads.s3.amazonaws.com%2Fuploads%2Fuser%2Fprofile_image%2F2533524%2F63d54c29-49fc-4cf7-8a18-18a098084828.png",
						ProfileURL: "https://dev.to/ahmedraza_fyntune",
					},
					Snippet:   "In the previous post, we built a simple CRUD API in Go using in-memory storage. Now let\u2019s make it...",
					Tag:       news.TagArticle,
					Comments:  0,
					Score:     0.227670248696953, // 1 reaction: log(2)/log(21); weight 1.0
					Published: time.Date(2026, time.April, 27, 11, 9, 38, 0, time.UTC),
				}, items[0])
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			got, err := DevTo{url: s.URL}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
