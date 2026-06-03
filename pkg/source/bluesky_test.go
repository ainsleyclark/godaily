// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBluesky_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/bluesky.json")
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
				// Fixture has 3 posts with like counts 5, 31, and 1.
				// The third is filtered out by the >=3 threshold.
				require.Len(t, items, 2)
				assert.Equal(t, news.SourceBluesky, items[0].Source)
				assert.Equal(t, news.TagSocial, items[0].Tag)
				assert.Equal(t,
					"https://bsky.app/profile/matt.bsky.social/post/3labcdef2x21",
					items[0].URL)
				assert.Equal(t, &news.Author{
					Name:       "Matt Boyle",
					Username:   "matt.bsky.social",
					AvatarURL:  "https://cdn.bsky.app/img/avatar/plain/did:plc:abc123/avatar@jpeg",
					ProfileURL: "https://bsky.app/profile/matt.bsky.social",
				}, items[0].Author)
				assert.Equal(t, 4, items[0].Comments)
				assert.Equal(t,
					news.ScoreOf(news.SourceBluesky, news.TagSocial, 5, true),
					items[0].Score)
				assert.NotEmpty(t, items[0].Title)
				// Second post carries an image embed that becomes ImageURL.
				assert.Equal(t,
					"https://cdn.bsky.app/img/feed_fullsize/plain/did:plc:def456/bench@jpeg",
					items[1].ImageURL)
			},
		},
		"Non-English language filtered": {
			body: []byte(`{"posts":[{"uri":"at://did:plc:x/app.bsky.feed.post/3xx","author":{"handle":"x.bsky.social","displayName":"X"},"record":{"text":"Schöne neue Go-Bibliothek #golang","createdAt":"2026-05-01T00:00:00.000Z","langs":["de"]},"replyCount":0,"likeCount":10}]}`),
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, items)
			},
		},
		"Empty text filtered": {
			body: []byte(`{"posts":[{"uri":"at://did:plc:x/app.bsky.feed.post/3xx","author":{"handle":"x.bsky.social","displayName":"X"},"record":{"text":"   ","createdAt":"2026-05-01T00:00:00.000Z","langs":["en"]},"replyCount":0,"likeCount":50}]}`),
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, items)
			},
		},
		"No language field kept": {
			body: []byte(`{"posts":[{"uri":"at://did:plc:x/app.bsky.feed.post/3xx","author":{"handle":"x.bsky.social","displayName":"X"},"record":{"text":"Go generics deep dive","createdAt":"2026-05-01T00:00:00.000Z"},"replyCount":0,"likeCount":4}]}`),
			want: func(t *testing.T, items []news.Item, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, news.SourceBluesky, items[0].Source)
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

			got, err := Bluesky{url: s.URL}.Fetch(t.Context())
			test.want(t, got, err)
		})
	}
}

func TestBlueskyPostURL(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		handle string
		uri    string
		want   string
	}{
		"ok": {
			handle: "matt.bsky.social",
			uri:    "at://did:plc:abc/app.bsky.feed.post/3labc",
			want:   "https://bsky.app/profile/matt.bsky.social/post/3labc",
		},
		"missing handle": {
			handle: "",
			uri:    "at://did:plc:abc/app.bsky.feed.post/3labc",
			want:   "",
		},
		"not a feed post": {
			handle: "matt.bsky.social",
			uri:    "at://did:plc:abc/app.bsky.feed.like/3labc",
			want:   "",
		},
		"empty rkey": {
			handle: "matt.bsky.social",
			uri:    "at://did:plc:abc/app.bsky.feed.post/",
			want:   "",
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, blueskyPostURL(tc.handle, tc.uri))
		})
	}
}
