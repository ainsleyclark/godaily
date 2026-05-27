// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package linkedin

import (
	"context"
	"encoding/json"
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

	c := New("tok", "urn:li:organization:1")
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

		c := New("my-token", "urn:li:organization:99")
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

		c := New("bad", "urn:li:organization:1")
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

		c := New("tok", "urn:li:organization:1")
		c.baseURL = srv.URL

		res, err := c.Post(context.Background(), platform.PostRequest{Text: "x"})
		require.NoError(t, err)
		assert.Empty(t, res.PostURL)
	})

	t.Run("Transport error", func(t *testing.T) {
		t.Parallel()

		c := New("tok", "urn")
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

		c := New("tok", "urn:li:organization:99")
		c.baseURL = srv.URL

		_, err := c.Post(context.Background(), platform.PostRequest{
			Text:               "Today we thank Ardan Labs for their writing.",
			MentionURN:         "urn:li:organization:42",
			MentionDisplayName: "Ardan Labs",
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

		c := New("tok", "urn:li:organization:99")
		c.baseURL = srv.URL

		_, err := c.Post(context.Background(), platform.PostRequest{
			Text:               "Today we thank some other source.",
			MentionURN:         "urn:li:organization:42",
			MentionDisplayName: "Ardan Labs",
		})
		require.NoError(t, err)
	})
}

func TestBuildAnnotations(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		text, urn, name string
		wantLen         int
		wantStart       int
	}{
		"Match at start":      {"Ardan Labs writes Go.", "urn:li:organization:1", "Ardan Labs", 1, 0},
		"Match mid-text":      {"Today, Ardan Labs ships.", "urn:li:organization:1", "Ardan Labs", 1, 7},
		"Case mismatch drops": {"ardan labs ships.", "urn:li:organization:1", "Ardan Labs", 0, 0},
		"Empty URN":           {"Ardan Labs ships.", "", "Ardan Labs", 0, 0},
		"Empty name":          {"Ardan Labs ships.", "urn:li:organization:1", "", 0, 0},
		"Empty text":          {"", "urn:li:organization:1", "Ardan Labs", 0, 0},
		"Not present":         {"Hello world.", "urn:li:organization:1", "Ardan Labs", 0, 0},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := buildAnnotations(test.text, test.urn, test.name)
			assert.Len(t, got, test.wantLen)
			if test.wantLen > 0 {
				assert.Equal(t, test.wantStart, got[0].Start)
				assert.Equal(t, len(test.name), got[0].Length)
				assert.Equal(t, test.urn, got[0].Entity)
			}
		})
	}
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
