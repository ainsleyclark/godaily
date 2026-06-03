// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkedIn_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("Not configured", func(t *testing.T) {
		t.Parallel()
		items, err := (&LinkedIn{url: "http://unused", client: linkedInNoRedirectClient}).Fetch(t.Context())
		assert.ErrorContains(t, err, "LINKEDIN_COOKIE is not set")
		assert.Nil(t, items)
	})

	t.Run("Missing JSESSIONID", func(t *testing.T) {
		t.Parallel()
		items, err := (&LinkedIn{url: "http://unused", cookie: "li_at=something", client: linkedInNoRedirectClient}).Fetch(t.Context())
		assert.ErrorContains(t, err, "JSESSIONID not found")
		assert.Nil(t, items)
	})

	fixture, err := os.ReadFile("testdata/linkedin.json")
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
				// Fixture has 3 posts: likes 87, 43, and 2.
				// Third is filtered by the >= linkedInMinLikes threshold.
				require.Len(t, items, 2)
				assert.Equal(t, news.SourceLinkedIn, items[0].Source)
				assert.Equal(t, news.TagSocial, items[0].Tag)
				// UTM params must be stripped from the share URL.
				assert.Equal(t, "https://www.linkedin.com/posts/alice-gopher_golang-activity-7234567890123456001-abcd", items[0].URL)
				// miniProfileUrn query param must be stripped from the profile URL.
				assert.Equal(t, &news.Author{
					Name:       "Alice Gopher",
					Username:   "Senior Go Engineer at CloudCo",
					ProfileURL: "https://www.linkedin.com/in/alice-gopher",
				}, items[0].Author)
				assert.NotEmpty(t, items[0].Title)
				assert.Equal(t,
					news.ScoreOf(news.SourceLinkedIn, news.TagSocial, 87, true),
					items[0].Score)
			},
		},
		"Low likes filtered": {
			body: []byte(`{"included":[{"$type":"com.linkedin.voyager.dash.feed.Update","entityUrn":"urn:li:fsd_update:1","metadata":{"backendUrn":"urn:li:activity:1"},"actor":{"name":{"text":"X"},"description":{"text":""},"navigationContext":{"actionTarget":""}},"commentary":{"text":{"text":"golang post"}},"socialContent":{"shareUrl":"https://www.linkedin.com/posts/x-activity-1-abcd"}},{"$type":"com.linkedin.voyager.dash.feed.SocialActivityCounts","entityUrn":"urn:li:fsd_socialActivityCounts:urn:li:activity:1","numLikes":3,"numComments":0}]}`),
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

			got, err := (&LinkedIn{url: s.URL, cookie: "li_at=test; JSESSIONID=ajax:123", client: linkedInNoRedirectClient}).Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestLinkedInTitle(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"plain":           {in: "Hello world", want: "Hello world"},
		"truncates":       {in: strings.Repeat("a", 200), want: strings.Repeat("a", 80)},
		"first sentence":  {in: "First sentence. Second sentence.", want: "First sentence"},
		"strips entities": {in: "foo &amp; bar", want: "foo & bar"},
		"newline split":   {in: "First line\nSecond line", want: "First line"},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, linkedInTitle(tc.in))
		})
	}
}
