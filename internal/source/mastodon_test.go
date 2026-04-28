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

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mastodonReblogResponse exercises the reblog filter: a status whose `reblog`
// field is non-null is a boost and must be dropped.
const mastodonReblogResponse = `[
  {
    "id": "1",
    "created_at": "2026-04-28T08:00:00.000Z",
    "url": "https://mastodon.social/@bot/1",
    "content": "<p>boosted post</p>",
    "replies_count": 0,
    "favourites_count": 50,
    "reblog": {"id": "x", "url": "https://other/", "content": "<p>orig</p>"},
    "account": {"username": "bot", "display_name": "Bot"},
    "media_attachments": []
  }
]`

func TestMastodon_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/mastodon.json")
	require.NoError(t, err)

	tt := map[string]struct {
		body []byte
		want func(t *testing.T, items []news.Item, err error)
	}{
		"Bad Request": {
			body: nil,
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				assert.Error(t, err)
				assert.Nil(t, items)
			},
		},
		"OK": {
			body: fixture,
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				// Fixture has 3 statuses: favourites_count of 5, 7, and 0.
				// The third is filtered out by the >=3 threshold.
				require.Len(t, items, 2)
				assert.Equal(t, news.SourceMastodon, items[0].Source)
				assert.Equal(t, "https://mastodon.world/@jobsfordevelopers/116481296525884754", items[0].URL)
				assert.Equal(t, "Jobs for Developers", items[0].Author)
				assert.Equal(t, news.TagArticle, items[0].Tag)
				assert.Equal(t,
					news.ScoreOf(news.SourceMastodon, news.TagArticle, 5, true),
					items[0].Score)
				assert.NotEmpty(t, items[0].Title)
				// Second item carries an image attachment that becomes ImageURL.
				assert.Equal(t, "https://example.com/img.jpg", items[1].ImageURL)
			},
		},
		"Reblog Dropped": {
			body: []byte(mastodonReblogResponse),
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, items)
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.body == nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(test.body)
			}))
			defer s.Close()

			got, err := Mastodon{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestMastodonTitle(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"plain":           {in: "<p>Hello world</p>", want: "Hello world"},
		"truncates":       {in: "<p>" + repeat("a", 200) + "</p>", want: repeat("a", 80)},
		"first sentence":  {in: "<p>First sentence. Second sentence.</p>", want: "First sentence"},
		"strips entities": {in: "<p>foo &amp; bar</p>", want: "foo & bar"},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, mastodonTitle(tc.in))
		})
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
