// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

func TestDevTo_Fetch(t *testing.T) {
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
			stub: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			want: func(items []news.Item, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "unexpected status code")
				assert.Nil(t, items)
			},
		},
		"Decode Error": {
			stub: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`bad json`))
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
					Score:     0,
					Tag:       news.TagProposal,
					Comments:  3,
					Published: time.Date(2026, time.April, 25, 11, 4, 19, 0, time.UTC),
				}, items[0])
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

			c := DevTo{
				http: s.Client(),
				url:  url,
			}

			got, err := c.Fetch(t.Context())
			test.want(got, err)
		})
	}

	t.Run("Do Error", func(t *testing.T) {
		f := NewDevTo()
		f.http = &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			}),
		}

		_, err := f.Fetch(t.Context())
		assert.ErrorContains(t, err, "fetch dev to")
	})
}

// roundTripFunc is a helper type to create a custom RoundTripper for testing
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
