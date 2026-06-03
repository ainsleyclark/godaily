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

// blueskySession is the canned createSession response used by the test server.
const blueskySession = `{"accessJwt":"test-token","refreshJwt":"r","handle":"godaily.bsky.social","did":"did:plc:test"}`

// newBlueskyTestClient builds a Bluesky source pointed at a test server that
// serves a canned session on createSession and the supplied body on
// searchPosts. sessionStatus lets a test force a session failure.
func newBlueskyTestServer(t *testing.T, searchBody []byte, sessionStatus int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "createSession"):
			if sessionStatus != http.StatusOK {
				w.WriteHeader(sessionStatus)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(blueskySession))
		default: // searchPosts
			if searchBody == nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(searchBody)
		}
	}))
}

func blueskySource(s *httptest.Server) Bluesky {
	return Bluesky{
		sessionURL:  s.URL + "/xrpc/com.atproto.server.createSession",
		searchURL:   s.URL + "/xrpc/app.bsky.feed.searchPosts",
		handle:      "godaily.bsky.social",
		appPassword: "app-pw",
		client:      s.Client(),
	}
}

func TestBluesky_Fetch(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/bluesky.json")
	require.NoError(t, err)

	t.Run("Missing credentials", func(t *testing.T) {
		t.Parallel()
		got, err := Bluesky{}.Fetch(t.Context())
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("Session failure", func(t *testing.T) {
		t.Parallel()
		s := newBlueskyTestServer(t, fixture, http.StatusUnauthorized)
		defer s.Close()
		got, err := blueskySource(s).Fetch(t.Context())
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("Search Bad Request", func(t *testing.T) {
		t.Parallel()
		s := newBlueskyTestServer(t, nil, http.StatusOK)
		defer s.Close()
		got, err := blueskySource(s).Fetch(t.Context())
		assert.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("OK", func(t *testing.T) {
		t.Parallel()
		s := newBlueskyTestServer(t, fixture, http.StatusOK)
		defer s.Close()
		items, err := blueskySource(s).Fetch(t.Context())
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
	})

	t.Run("Non-English language filtered", func(t *testing.T) {
		t.Parallel()
		body := []byte(`{"posts":[{"uri":"at://did:plc:x/app.bsky.feed.post/3xx","author":{"handle":"x.bsky.social","displayName":"X"},"record":{"text":"Schöne neue Go-Bibliothek #golang","createdAt":"2026-05-01T00:00:00.000Z","langs":["de"]},"replyCount":0,"likeCount":10}]}`)
		s := newBlueskyTestServer(t, body, http.StatusOK)
		defer s.Close()
		items, err := blueskySource(s).Fetch(t.Context())
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("Empty text filtered", func(t *testing.T) {
		t.Parallel()
		body := []byte(`{"posts":[{"uri":"at://did:plc:x/app.bsky.feed.post/3xx","author":{"handle":"x.bsky.social","displayName":"X"},"record":{"text":"   ","createdAt":"2026-05-01T00:00:00.000Z","langs":["en"]},"replyCount":0,"likeCount":50}]}`)
		s := newBlueskyTestServer(t, body, http.StatusOK)
		defer s.Close()
		items, err := blueskySource(s).Fetch(t.Context())
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("No language field kept", func(t *testing.T) {
		t.Parallel()
		body := []byte(`{"posts":[{"uri":"at://did:plc:x/app.bsky.feed.post/3xx","author":{"handle":"x.bsky.social","displayName":"X"},"record":{"text":"Go generics deep dive","createdAt":"2026-05-01T00:00:00.000Z"},"replyCount":0,"likeCount":4}]}`)
		s := newBlueskyTestServer(t, body, http.StatusOK)
		defer s.Close()
		items, err := blueskySource(s).Fetch(t.Context())
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, news.SourceBluesky, items[0].Source)
	})
}

func TestBlueskyTitle(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		in   string
		want string
	}{
		"plain":             {in: "Go generics deep dive", want: "Go generics deep dive"},
		"keeps version":     {in: "🎆 Go 1.26.4 and 1.25.11 are released!\n\n🔐 Security fixes", want: "🎆 Go 1.26.4 and 1.25.11 are released"},
		"keeps dotted name": {in: "yzma 1.15 is out for llama.cpp users", want: "yzma 1.15 is out for llama.cpp users"},
		"first sentence":    {in: "First sentence. Second sentence.", want: "First sentence"},
		"first line":        {in: "Title line\nbody text here", want: "Title line"},
		"truncates":         {in: repeat("a", 200), want: repeat("a", 80)},
		"trailing question": {in: "Anyone using sqlc?", want: "Anyone using sqlc"},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, blueskyTitle(tc.in))
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
