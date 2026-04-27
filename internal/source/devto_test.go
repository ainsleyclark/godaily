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

func TestDevTo_Fetch(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		stub http.HandlerFunc
		url  string
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
				_, err := w.Write([]byte(`[{"type_of":"article","id":3549510,"title":"I built agent-to-agent communication that works behind any NAT","description":"If you have ever tried connecting two AI agents running on different machines...","slug":"slug","path":"/artem_a/slug","url":"https://dev.to/artem_a/slug","comments_count":3,"public_reactions_count":0,"collection_id":null,"published_timestamp":"2026-04-25T11:04:19Z","language":"en","subforem_id":1,"positive_reactions_count":0,"cover_image":null,"social_image":"","canonical_url":"https://dev.to/artem_a/slug","created_at":"2026-04-25T11:04:19Z","edited_at":null,"crossposted_at":null,"published_at":"2026-04-25T11:04:19Z","last_comment_at":"2026-04-25T11:04:19Z","reading_time_minutes":4,"tag_list":["go"],"tags":"go","user":{"name":"Artemii Amelin","username":"artem_a","twitter_username":null,"github_username":"artemiia","user_id":3893832,"website_url":null,"profile_image":"","profile_image_90":""}}]`))
				assert.NoError(t, err)
			},
			want: func(items []news.Item, err error) {
				assert.NoError(t, err)
				assert.Len(t, items, 1)
				assert.Equal(t, news.Item{
					Source:    news.SourceDevTo,
					Title:     "I built agent-to-agent communication that works behind any NAT",
					URL:       "https://dev.to/artem_a/slug",
					Author:    "Artemii Amelin",
					Snippet:   "If you have ever tried connecting two AI agents running on different machines...",
					Tag:       news.TagProposal,
					Comments:  3,
					Published: time.Date(2026, time.April, 25, 11, 4, 19, 0, time.UTC),
				}, items[0])
			},
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(test.stub)
			defer s.Close()

			url := s.URL
			if test.url != "" {
				url = test.url
			}

			got, err := DevTo{url: url}.Fetch(t.Context())
			test.want(got, err)
		})
	}
}
