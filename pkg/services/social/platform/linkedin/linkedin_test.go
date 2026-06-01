// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package linkedin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

func TestClient_Platform(t *testing.T) {
	t.Parallel()

	c := New("tok", "urn:li:organization:1", "")
	assert.Equal(t, social.LinkedIn, c.Platform())
}

func TestClient_Post(t *testing.T) {
	t.Parallel()

	t.Run("Happy path returns feed URL from x-restli-id", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rest/posts", r.URL.Path)
			assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, defaultAPIVersion, r.Header.Get("LinkedIn-Version"))
			assert.Equal(t, "2.0.0", r.Header.Get("X-Restli-Protocol-Version"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(body, &got))
			assert.Equal(t, "urn:li:organization:99", got["author"])
			assert.Equal(t, "Hello, Go community", got["commentary"])
			assert.Equal(t, "PUBLIC", got["visibility"])
			assert.Equal(t, "PUBLISHED", got["lifecycleState"])

			w.Header().Set("x-restli-id", "urn:li:share:7234567890123456789")
			w.WriteHeader(http.StatusCreated)
		}))
		defer srv.Close()

		c := New("my-token", "urn:li:organization:99", "")
		c.baseURL = srv.URL

		res, err := c.Post(context.Background(), platform.PostRequest{Text: "Hello, Go community"})
		require.NoError(t, err)
		assert.Equal(
			t,
			"https://www.linkedin.com/feed/update/urn:li:share:7234567890123456789/",
			res.PostURL,
		)
	})

	t.Run("Non-2xx surfaces body in error", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"invalid token"}`))
		}))
		defer srv.Close()

		c := New("bad", "urn:li:organization:1", "")
		c.baseURL = srv.URL

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("Missing x-restli-id yields empty URL", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", "")
		c.baseURL = srv.URL

		res, err := c.Post(context.Background(), platform.PostRequest{Text: "x"})
		require.NoError(t, err)
		assert.Empty(t, res.PostURL)
	})

	t.Run("Transport error", func(t *testing.T) {
		t.Parallel()

		c := New("tok", "urn", "")
		c.baseURL = "http://127.0.0.1:1"

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "x"})
		require.Error(t, err)
	})
}

func TestClient_Post_Annotation(t *testing.T) {
	t.Parallel()

	t.Run("Mention with matching display name produces annotation", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(body, &got))

			anns, ok := got["commentaryAnnotations"].([]any)
			require.True(t, ok, "expected commentaryAnnotations array")
			require.Len(t, anns, 1)
			a := anns[0].(map[string]any)
			assert.Equal(t, float64(15), a["start"])
			assert.Equal(t, float64(10), a["length"])
			assert.Equal(t, "urn:li:organization:42", a["entity"])

			w.Header().Set("x-restli-id", "urn:li:share:1")
			w.WriteHeader(http.StatusCreated)
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:99", "")
		c.baseURL = srv.URL

		_, err := c.Post(context.Background(), platform.PostRequest{
			Text: "Today we thank Ardan Labs for their writing.",
			Mentions: []social.Mention{
				{Platform: social.LinkedIn, DisplayName: "Ardan Labs", Handle: "urn:li:organization:42"},
			},
		})
		require.NoError(t, err)
	})

	t.Run("Missing display name in text omits annotations", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var got map[string]any
			require.NoError(t, json.Unmarshal(body, &got))
			_, present := got["commentaryAnnotations"]
			assert.False(t, present, "commentaryAnnotations should be omitted when no match")

			w.Header().Set("x-restli-id", "urn:li:share:1")
			w.WriteHeader(http.StatusCreated)
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:99", "")
		c.baseURL = srv.URL

		_, err := c.Post(context.Background(), platform.PostRequest{
			Text: "Today we thank some other source.",
			Mentions: []social.Mention{
				{Platform: social.LinkedIn, DisplayName: "Ardan Labs", Handle: "urn:li:organization:42"},
			},
		})
		require.NoError(t, err)
	})
}

func TestClient_Stats(t *testing.T) {
	t.Parallel()

	t.Run("Happy path returns engagement counts", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rest/organizationalEntityShareStatistics", r.URL.Path)
			assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
			assert.Equal(t, "2.0.0", r.Header.Get("X-Restli-Protocol-Version"))

			q := r.URL.Query()
			assert.Equal(t, "organizationalEntity", q.Get("q"))
			assert.Equal(t, "urn:li:organization:99", q.Get("organizationalEntity"))
			// Rest.li 2.0 array encoding — the legacy shares[0]= form is
			// rejected with 400 QUERY_PARAM_NOT_ALLOWED.
			assert.Equal(t, "List(urn:li:share:7234567890)", q.Get("shares"))
			assert.Empty(t, q.Get("shares[0]"))

			_, _ = w.Write([]byte(`{"elements":[{"totalShareStatistics":{` +
				`"likeCount":19,"commentCount":4,"shareCount":5,"impressionCount":400,"clickCount":7}}]}`))
		}))
		defer srv.Close()

		c := New("my-token", "urn:li:organization:99", "")
		c.baseURL = srv.URL

		got, err := c.Stats(
			context.Background(),
			"https://www.linkedin.com/feed/update/urn:li:share:7234567890/",
		)
		require.NoError(t, err)
		assert.Equal(t, platform.Stats{Likes: 19, Reposts: 5, Comments: 4, Impressions: 400}, got)
	})

	t.Run("No elements yields zero stats", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"elements":[]}`))
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", "")
		c.baseURL = srv.URL

		got, err := c.Stats(context.Background(), "https://www.linkedin.com/feed/update/urn:li:share:1/")
		require.NoError(t, err)
		assert.Equal(t, platform.Stats{}, got)
	})

	t.Run("Non-2xx surfaces body in error", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"Invalid param"}`))
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", "")
		c.baseURL = srv.URL

		_, err := c.Stats(context.Background(), "https://www.linkedin.com/feed/update/urn:li:share:1/")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
		assert.Contains(t, err.Error(), "Invalid param")
	})

	t.Run("Deleted post surfaces ErrPostUnavailable", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"Unable to get activityIds from any of the given shares. Either the shares/ugcPosts do not have corresponding activities or the organizational entity did not post them.","status":400}`))
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", "")
		c.baseURL = srv.URL

		_, err := c.Stats(context.Background(), "https://www.linkedin.com/feed/update/urn:li:share:1/")
		require.Error(t, err)
		assert.True(t, errors.Is(err, platform.ErrPostUnavailable))
	})

	t.Run("Malformed URL errors before request", func(t *testing.T) {
		t.Parallel()

		c := New("tok", "urn:li:organization:1", "")
		c.baseURL = "http://127.0.0.1:1"

		_, err := c.Stats(context.Background(), "https://www.linkedin.com/feed/update/not-a-urn/")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "share URN")
	})
}

func TestBuildAnnotations(t *testing.T) {
	t.Parallel()

	lin := func(displayName, handle string) social.Mention {
		return social.Mention{Platform: social.LinkedIn, DisplayName: displayName, Handle: handle}
	}

	t.Run("Single mention matched", func(t *testing.T) {
		t.Parallel()
		got, missed := buildAnnotations(
			"Ardan Labs writes Go.",
			[]social.Mention{lin("Ardan Labs", "urn:li:organization:1")},
		)
		require.Len(t, got, 1)
		assert.Equal(t, 0, got[0].Start)
		assert.Equal(t, 10, got[0].Length)
		assert.Equal(t, "urn:li:organization:1", got[0].Entity)
		assert.Empty(t, missed)
	})

	t.Run("Two non-overlapping mentions both matched in document order", func(t *testing.T) {
		t.Parallel()
		got, missed := buildAnnotations(
			"Today Ardan Labs and William Kennedy ship.",
			[]social.Mention{
				lin("William Kennedy", "urn:li:person:2"),
				lin("Ardan Labs", "urn:li:organization:1"),
			},
		)
		require.Len(t, got, 2)
		assert.Equal(t, "urn:li:organization:1", got[0].Entity, "Ardan Labs appears first in text")
		assert.Equal(t, "urn:li:person:2", got[1].Entity)
		assert.Empty(t, missed)
	})

	t.Run("Overlapping mentions: longer wins", func(t *testing.T) {
		t.Parallel()
		// "Go" is a substring of "Go Blog"; the longer match wins, the
		// shorter mention is dropped as missed.
		got, _ := buildAnnotations(
			"The Go Blog covers Go internals.",
			[]social.Mention{
				lin("Go", "urn:li:organization:99"),
				lin("Go Blog", "urn:li:organization:1"),
			},
		)
		require.Len(t, got, 1)
		assert.Equal(t, "urn:li:organization:1", got[0].Entity)
		assert.Equal(t, 7, got[0].Length)
	})

	t.Run("Missing display name surfaced in missed list", func(t *testing.T) {
		t.Parallel()
		got, missed := buildAnnotations(
			"Hello world.",
			[]social.Mention{lin("Ardan Labs", "urn:li:organization:1")},
		)
		assert.Empty(t, got)
		require.Len(t, missed, 1)
		assert.Equal(t, "Ardan Labs", missed[0].DisplayName)
	})

	t.Run("Non-LinkedIn mentions ignored", func(t *testing.T) {
		t.Parallel()
		got, missed := buildAnnotations(
			"Ardan Labs writes Go.",
			[]social.Mention{
				{Platform: social.Bluesky, DisplayName: "Ardan Labs", Handle: "@ardanlabs.com"},
				{Platform: social.Mastodon, DisplayName: "Ardan Labs", Handle: "@ardanlabs@hachyderm.io"},
			},
		)
		assert.Empty(t, got)
		assert.Empty(t, missed)
	})

	t.Run("Empty inputs return nil", func(t *testing.T) {
		t.Parallel()
		got, missed := buildAnnotations("", []social.Mention{lin("X", "urn:li:organization:1")})
		assert.Nil(t, got)
		assert.Nil(t, missed)

		got, missed = buildAnnotations("Text", nil)
		assert.Nil(t, got)
		assert.Nil(t, missed)
	})

	t.Run("Empty handle or display name skipped silently", func(t *testing.T) {
		t.Parallel()
		got, missed := buildAnnotations(
			"Ardan Labs writes Go.",
			[]social.Mention{
				{Platform: social.LinkedIn, DisplayName: "", Handle: "urn:li:organization:1"},
				{Platform: social.LinkedIn, DisplayName: "Ardan Labs", Handle: ""},
			},
		)
		assert.Empty(t, got)
		assert.Empty(t, missed)
	})
}

func TestClient_HasLiked(t *testing.T) {
	t.Parallel()

	const memberID = "abc123EncodedId"
	const postURL = "https://www.linkedin.com/feed/update/urn:li:share:7234567890/"

	t.Run("Returns true when member ID found in first page", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rest/reactions", r.URL.Path)
			assert.Equal(t, "entity", r.URL.Query().Get("q"))
			_, _ = w.Write([]byte(`{"paging":{"total":1},"elements":[{"actor":"urn:li:person:abc123EncodedId"}]}`))
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", memberID)
		c.baseURL = srv.URL

		got, err := c.HasLiked(context.Background(), postURL)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("Matches regardless of URN prefix", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"paging":{"total":1},"elements":[{"actor":"urn:li:fsd_profile:abc123EncodedId"}]}`))
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", memberID)
		c.baseURL = srv.URL

		got, err := c.HasLiked(context.Background(), postURL)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("Returns false when member ID not present", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"paging":{"total":1},"elements":[{"actor":"urn:li:person:someone-else"}]}`))
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", memberID)
		c.baseURL = srv.URL

		got, err := c.HasLiked(context.Background(), postURL)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("Returns false with no error when memberURN is empty", func(t *testing.T) {
		t.Parallel()

		c := New("tok", "urn:li:organization:1", "")
		got, err := c.HasLiked(context.Background(), postURL)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("Pages through results to find member", func(t *testing.T) {
		t.Parallel()

		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.URL.Query().Get("start") == "0" {
				_, _ = w.Write([]byte(`{"paging":{"total":2},"elements":[{"actor":"urn:li:person:other"}]}`))
			} else {
				_, _ = w.Write([]byte(`{"paging":{"total":2},"elements":[{"actor":"urn:li:person:abc123EncodedId"}]}`))
			}
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", memberID)
		c.baseURL = srv.URL

		got, err := c.HasLiked(context.Background(), postURL)
		require.NoError(t, err)
		assert.True(t, got)
		assert.Equal(t, 2, calls)
	})

	t.Run("Transport error surfaces as error", func(t *testing.T) {
		t.Parallel()

		c := New("tok", "urn:li:organization:1", memberID)
		c.baseURL = "http://127.0.0.1:1"

		_, err := c.HasLiked(context.Background(), postURL)
		require.Error(t, err)
	})
}

func TestClient_HasReposted(t *testing.T) {
	t.Parallel()

	const memberID = "abc123EncodedId"
	const postURL = "https://www.linkedin.com/feed/update/urn:li:share:7234567890/"

	t.Run("Returns true when member ID found in reshares", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rest/shares", r.URL.Path)
			assert.Equal(t, "sharesOfShare", r.URL.Query().Get("q"))
			_, _ = w.Write([]byte(`{"paging":{"total":1},"elements":[{"author":"urn:li:person:abc123EncodedId"}]}`))
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", memberID)
		c.baseURL = srv.URL

		got, err := c.HasReposted(context.Background(), postURL)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("Returns false when member ID not present", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"paging":{"total":1},"elements":[{"author":"urn:li:person:someone-else"}]}`))
		}))
		defer srv.Close()

		c := New("tok", "urn:li:organization:1", memberID)
		c.baseURL = srv.URL

		got, err := c.HasReposted(context.Background(), postURL)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("Returns false with no error when memberURN is empty", func(t *testing.T) {
		t.Parallel()

		c := New("tok", "urn:li:organization:1", "")
		got, err := c.HasReposted(context.Background(), postURL)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("Transport error surfaces as error", func(t *testing.T) {
		t.Parallel()

		c := New("tok", "urn:li:organization:1", memberID)
		c.baseURL = "http://127.0.0.1:1"

		_, err := c.HasReposted(context.Background(), postURL)
		require.Error(t, err)
	})
}

func TestFeedURL(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input string
		want  string
	}{
		"Happy path": {
			input: "urn:li:share:7234567890123456789",
			want:  "https://www.linkedin.com/feed/update/urn:li:share:7234567890123456789/",
		},
		"Empty": {
			input: "",
			want:  "",
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := feedURL(test.input)
			assert.Equal(t, test.want, got)
		})
	}
}
