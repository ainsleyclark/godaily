// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bluesky

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

func TestClient_Platform(t *testing.T) {
	t.Parallel()

	c := New("godaily.bsky.social", "pw")
	assert.Equal(t, social.Bluesky, c.Platform())
}

func TestClient_Post(t *testing.T) {
	t.Parallel()

	t.Run("Happy path returns post URL", func(t *testing.T) {
		t.Parallel()

		var sessionHits, recordHits int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))

			switch r.URL.Path {
			case "/xrpc/com.atproto.server.createSession":
				sessionHits++
				body, _ := io.ReadAll(r.Body)
				var got map[string]string
				require.NoError(t, json.Unmarshal(body, &got))
				assert.Equal(t, "godaily.bsky.social", got["identifier"])
				assert.Equal(t, "secret", got["password"])
				_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:xyz"}`))

			case "/xrpc/com.atproto.repo.createRecord":
				recordHits++
				assert.Equal(t, "Bearer jwt-token", r.Header.Get("Authorization"))
				body, _ := io.ReadAll(r.Body)
				var got map[string]any
				require.NoError(t, json.Unmarshal(body, &got))
				assert.Equal(t, "did:plc:xyz", got["repo"])
				assert.Equal(t, "app.bsky.feed.post", got["collection"])
				record, ok := got["record"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "Go 1.30 released", record["text"])
				_, _ = w.Write([]byte(`{"uri":"at://did:plc:xyz/app.bsky.feed.post/3kabcdef","cid":"bafy"}`))

			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		defer srv.Close()

		c := New("godaily.bsky.social", "secret")
		c.baseURL = srv.URL
		c.publicURL = "https://bsky.app"

		got, err := c.Post(context.Background(), platform.PostRequest{Text: "Go 1.30 released"})
		require.NoError(t, err)
		assert.Equal(t, "https://bsky.app/profile/godaily.bsky.social/post/3kabcdef", got.PostURL)
		assert.Equal(t, 1, sessionHits)
		assert.Equal(t, 1, recordHits)
	})

	t.Run("Session error surfaces", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"AuthenticationRequired"}`))
		}))
		defer srv.Close()

		c := New("h", "bad")
		c.baseURL = srv.URL

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "createSession")
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("CreateRecord error surfaces", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/xrpc/com.atproto.server.createSession":
				_, _ = w.Write([]byte(`{"accessJwt":"jwt","did":"did:plc:abc"}`))
			case "/xrpc/com.atproto.repo.createRecord":
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"InvalidRequest"}`))
			}
		}))
		defer srv.Close()

		c := New("h", "pw")
		c.baseURL = srv.URL

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "createRecord")
	})

	t.Run("Transport error", func(t *testing.T) {
		t.Parallel()

		c := New("h", "pw")
		c.baseURL = "http://127.0.0.1:1" // nothing listening

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "x"})
		require.Error(t, err)
	})

	t.Run("Facets included when text contains URL", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/xrpc/com.atproto.server.createSession":
				_, _ = w.Write([]byte(`{"accessJwt":"jwt","did":"did:plc:xyz"}`))
			case "/xrpc/com.atproto.repo.createRecord":
				body, _ := io.ReadAll(r.Body)
				var got map[string]any
				require.NoError(t, json.Unmarshal(body, &got))
				record, ok := got["record"].(map[string]any)
				require.True(t, ok)
				facets, ok := record["facets"]
				require.True(t, ok, "expected facets field in record")
				fl, ok := facets.([]any)
				require.True(t, ok)
				require.Len(t, fl, 1, "expected one facet for the URL")
				_, _ = w.Write([]byte(`{"uri":"at://did:plc:xyz/app.bsky.feed.post/abc","cid":"x"}`))
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		defer srv.Close()

		c := New("godaily.bsky.social", "pw")
		c.baseURL = srv.URL

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "New release\n\nhttps://go.dev/dl\n#golang"})
		require.NoError(t, err)
	})

	t.Run("Over-long text is capped to the grapheme limit", func(t *testing.T) {
		t.Parallel()

		var sentText string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/xrpc/com.atproto.server.createSession":
				_, _ = w.Write([]byte(`{"accessJwt":"jwt","did":"did:plc:xyz"}`))
			case "/xrpc/com.atproto.repo.createRecord":
				body, _ := io.ReadAll(r.Body)
				var got map[string]any
				require.NoError(t, json.Unmarshal(body, &got))
				record, ok := got["record"].(map[string]any)
				require.True(t, ok)
				sentText, _ = record["text"].(string)
				_, _ = w.Write([]byte(`{"uri":"at://did:plc:xyz/app.bsky.feed.post/abc","cid":"x"}`))
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		defer srv.Close()

		c := New("godaily.bsky.social", "pw")
		c.baseURL = srv.URL

		// 523 graphemes, mirroring the real "got 523" production failure.
		long := strings.Repeat("a", 523)
		_, err := c.Post(context.Background(), platform.PostRequest{Text: long})
		require.NoError(t, err)
		assert.LessOrEqual(t, utf8.RuneCountInString(sentText), maxGraphemes,
			"text sent to Bluesky must respect the grapheme limit")
	})
}

func TestClient_Stats(t *testing.T) {
	t.Parallel()

	t.Run("Resolves handle to DID then fetches engagement", func(t *testing.T) {
		t.Parallel()

		var resolveHits, getPostsHits int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/xrpc/com.atproto.identity.resolveHandle":
				resolveHits++
				assert.Equal(t, "godaily.dev", r.URL.Query().Get("handle"))
				_, _ = w.Write([]byte(`{"did":"did:plc:xyz"}`))

			case "/xrpc/app.bsky.feed.getPosts":
				getPostsHits++
				// The URI must use the resolved DID, not the handle.
				assert.Equal(t, "at://did:plc:xyz/app.bsky.feed.post/3kabcdef", r.URL.Query().Get("uris[]"))
				_, _ = w.Write([]byte(`{"posts":[{"likeCount":12,"repostCount":3,"replyCount":5}]}`))

			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		defer srv.Close()

		c := New("godaily.dev", "pw")
		c.appViewURL = srv.URL

		got, err := c.Stats(context.Background(), "https://bsky.app/profile/godaily.dev/post/3kabcdef")
		require.NoError(t, err)
		assert.Equal(t, platform.Stats{Likes: 12, Reposts: 3, Comments: 5}, got)
		assert.Equal(t, 1, resolveHits)
		assert.Equal(t, 1, getPostsHits)
	})

	t.Run("Empty posts array yields zero stats", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/xrpc/com.atproto.identity.resolveHandle":
				_, _ = w.Write([]byte(`{"did":"did:plc:xyz"}`))
			case "/xrpc/app.bsky.feed.getPosts":
				_, _ = w.Write([]byte(`{"posts":[]}`))
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		defer srv.Close()

		c := New("godaily.dev", "pw")
		c.appViewURL = srv.URL

		got, err := c.Stats(context.Background(), "https://bsky.app/profile/godaily.dev/post/abc")
		require.NoError(t, err)
		assert.Equal(t, platform.Stats{}, got)
	})

	t.Run("Malformed post URL errors before any request", func(t *testing.T) {
		t.Parallel()

		c := New("godaily.dev", "pw")
		c.appViewURL = "http://127.0.0.1:1" // must not be hit

		_, err := c.Stats(context.Background(), "https://bsky.app/not/a/post")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing post URL")
	})

	t.Run("Handle resolution failure surfaces", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"InvalidRequest"}`))
		}))
		defer srv.Close()

		c := New("godaily.dev", "pw")
		c.appViewURL = srv.URL

		_, err := c.Stats(context.Background(), "https://bsky.app/profile/godaily.dev/post/abc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolving handle to DID")
	})
}

func TestParsePostURL(t *testing.T) {
	t.Parallel()

	t.Run("Happy path", func(t *testing.T) {
		t.Parallel()
		handle, rKey, err := parsePostURL("https://bsky.app/profile/godaily.dev/post/3kabcdef")
		require.NoError(t, err)
		assert.Equal(t, "godaily.dev", handle)
		assert.Equal(t, "3kabcdef", rKey)
	})

	t.Run("Unexpected format errors", func(t *testing.T) {
		t.Parallel()
		_, _, err := parsePostURL("https://bsky.app/profile/godaily.dev")
		require.Error(t, err)
	})
}

func TestPostURLFromURI(t *testing.T) {
	t.Parallel()

	c := New("godaily.bsky.social", "pw")

	tt := map[string]struct {
		input string
		want  string
	}{
		"Happy path": {
			input: "at://did:plc:xyz/app.bsky.feed.post/3kabcdef",
			want:  "https://bsky.app/profile/godaily.bsky.social/post/3kabcdef",
		},
		"Missing prefix": {
			input: "did:plc:xyz/app.bsky.feed.post/3k",
			want:  "",
		},
		"Too few parts": {
			input: "at://did:plc:xyz",
			want:  "",
		},
		"Empty": {
			input: "",
			want:  "",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := c.postURLFromURI(test.input)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestBuildFacets(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		text string
		want []facet
	}{
		"No URL": {
			text: "Go 1.30 released",
			want: nil,
		},
		"Single URL": {
			text: "New release\n\nhttps://go.dev/dl\n#golang",
			want: []facet{{
				Index:    facetIndex{ByteStart: 13, ByteEnd: 30},
				Features: []facetFeature{{Type: "app.bsky.richtext.facet#link", URI: "https://go.dev/dl"}},
			}},
		},
		"URL with trailing period trimmed": {
			text: "See https://example.com.",
			want: []facet{{
				Index:    facetIndex{ByteStart: 4, ByteEnd: 23},
				Features: []facetFeature{{Type: "app.bsky.richtext.facet#link", URI: "https://example.com"}},
			}},
		},
		"Two URLs": {
			text: "https://a.io and https://b.io",
			want: []facet{
				{
					Index:    facetIndex{ByteStart: 0, ByteEnd: 12},
					Features: []facetFeature{{Type: "app.bsky.richtext.facet#link", URI: "https://a.io"}},
				},
				{
					Index:    facetIndex{ByteStart: 17, ByteEnd: 29},
					Features: []facetFeature{{Type: "app.bsky.richtext.facet#link", URI: "https://b.io"}},
				},
			},
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := buildFacets(tc.text)
			assert.Equal(t, tc.want, got)
		})
	}
}

// Sanity check that body is sent as application/json.
func TestClient_doJSON_SetsHeaders(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.True(t, strings.HasPrefix(r.URL.Path, "/xrpc/"))
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New("h", "p")
	c.baseURL = srv.URL

	err := c.doJSON(context.Background(), "x.y.z", "tok", map[string]string{"k": "v"}, nil)
	require.NoError(t, err)
}
