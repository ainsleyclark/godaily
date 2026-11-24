// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package linkedin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

// newRefreshServer returns a test server that serves both the OAuth token
// endpoint (/oauth) and /rest/posts. /rest/posts only accepts
// freshToken; /oauth always mints freshToken. Counters track how many times each was
// hit.
func newRefreshServer(t *testing.T, oauthStatus int) (srv *httptest.Server, posts, refreshes *atomic.Int32) {
	t.Helper()

	const validToken = freshToken
	posts = &atomic.Int32{}
	refreshes = &atomic.Int32{}

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth":
			refreshes.Add(1)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			form, err := url.ParseQuery(string(body))
			require.NoError(t, err)
			assert.Equal(t, "refresh_token", form.Get("grant_type"))
			assert.Equal(t, "refresh-tok", form.Get("refresh_token"))
			assert.Equal(t, "client-id", form.Get("client_id"))
			assert.Equal(t, "client-secret", form.Get("client_secret"))

			if oauthStatus != http.StatusOK {
				w.WriteHeader(oauthStatus)
				_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"` + validToken + `","expires_in":5184000,"refresh_token_expires_in":31000000}`))
		case "/rest/posts":
			posts.Add(1)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), `"commentary":"hi"`, "body must be resent intact on retry")

			if r.Header.Get("Authorization") != "Bearer "+validToken {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"status":401,"code":"EXPIRED_ACCESS_TOKEN"}`))
				return
			}
			w.Header().Set("x-restli-id", "urn:li:share:1")
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, posts, refreshes
}

const freshToken = "fresh"

func newRefreshClient(srv *httptest.Server, token string) *Client {
	c := New(token, "urn:li:organization:1", "", WithRefresh("refresh-tok", "client-id", "client-secret"))
	c.baseURL = srv.URL
	c.refresh.oauthURL = srv.URL + "/oauth"
	return c
}

func TestWithRefresh(t *testing.T) {
	t.Parallel()

	t.Run("No-op when any credential is missing", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, New("tok", "urn", "", WithRefresh("", "id", "secret")).refresh)
		assert.Nil(t, New("tok", "urn", "", WithRefresh("rt", "", "secret")).refresh)
		assert.Nil(t, New("tok", "urn", "", WithRefresh("rt", "id", "")).refresh)
		assert.NotNil(t, New("tok", "urn", "", WithRefresh("rt", "id", "secret")).refresh)
	})
}

func TestClient_Refresh(t *testing.T) {
	t.Parallel()

	t.Run("Expired token is refreshed and request retried once", func(t *testing.T) {
		t.Parallel()

		srv, posts, refreshes := newRefreshServer(t, http.StatusOK)
		c := newRefreshClient(srv, "expired")

		res, err := c.Post(context.Background(), platform.PostRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, "https://www.linkedin.com/feed/update/urn:li:share:1/", res.PostURL)
		assert.Equal(t, int32(2), posts.Load())
		assert.Equal(t, int32(1), refreshes.Load())

		// The refreshed token is reused without another refresh.
		_, err = c.Post(context.Background(), platform.PostRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, int32(3), posts.Load())
		assert.Equal(t, int32(1), refreshes.Load())
	})

	t.Run("Valid token never triggers a refresh", func(t *testing.T) {
		t.Parallel()

		srv, posts, refreshes := newRefreshServer(t, http.StatusOK)
		c := newRefreshClient(srv, "fresh")

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, int32(1), posts.Load())
		assert.Equal(t, int32(0), refreshes.Load())
	})

	t.Run("Missing access token is fetched up front", func(t *testing.T) {
		t.Parallel()

		srv, posts, refreshes := newRefreshServer(t, http.StatusOK)
		c := newRefreshClient(srv, "")

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "hi"})
		require.NoError(t, err)
		assert.Equal(t, int32(1), posts.Load())
		assert.Equal(t, int32(1), refreshes.Load())
	})

	t.Run("Refresh failure is surfaced", func(t *testing.T) {
		t.Parallel()

		srv, posts, refreshes := newRefreshServer(t, http.StatusBadRequest)
		c := newRefreshClient(srv, "expired")

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "hi"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token refresh")
		assert.Contains(t, err.Error(), "invalid_grant")
		assert.NotContains(t, err.Error(), "client-secret")
		assert.Equal(t, int32(1), posts.Load())
		assert.Equal(t, int32(1), refreshes.Load())
	})

	t.Run("Without refresh credentials a 401 is returned as-is", func(t *testing.T) {
		t.Parallel()

		srv, posts, refreshes := newRefreshServer(t, http.StatusOK)
		c := New("expired", "urn:li:organization:1", "")
		c.baseURL = srv.URL

		_, err := c.Post(context.Background(), platform.PostRequest{Text: "hi"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "EXPIRED_ACCESS_TOKEN")
		assert.Equal(t, int32(1), posts.Load())
		assert.Equal(t, int32(0), refreshes.Load())
	})
}
