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

	"github.com/ainsleyclark/godaily/pkg/gateway/social"
)

func TestClient_Platform(t *testing.T) {
	t.Parallel()

	c := New("tok", "urn:li:organization:1")
	assert.Equal(t, social.PlatformLinkedIn, c.Platform())
}

func TestClient_Post(t *testing.T) {
	t.Parallel()

	t.Run("Happy path returns feed URL from x-restli-id", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rest/posts", r.URL.Path)
			assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "202504", r.Header.Get("LinkedIn-Version"))
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

		res, err := c.Post(context.Background(), "Hello, Go community")
		require.NoError(t, err)
		assert.Equal(t,
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

		_, err := c.Post(context.Background(), "x")
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

		res, err := c.Post(context.Background(), "x")
		require.NoError(t, err)
		assert.Empty(t, res.PostURL)
	})

	t.Run("Transport error", func(t *testing.T) {
		t.Parallel()

		c := New("tok", "urn")
		c.baseURL = "http://127.0.0.1:1"

		_, err := c.Post(context.Background(), "x")
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
