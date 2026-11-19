// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPAuth(t *testing.T) {
	t.Parallel()

	const (
		mcpKey = "mcp-read-only"
		admin  = "admin-secret"
	)

	tt := map[string]struct {
		mcpKey, admin string
		target        string
		headers       map[string]string
		wantStatus    int
	}{
		"MCP key as bearer": {
			mcpKey: mcpKey, admin: admin,
			headers:    map[string]string{"Authorization": "Bearer " + mcpKey},
			wantStatus: http.StatusOK,
		},
		"MCP key as X-API-Key": {
			mcpKey: mcpKey, admin: admin,
			headers:    map[string]string{"X-API-Key": mcpKey},
			wantStatus: http.StatusOK,
		},
		"MCP key in query": {
			mcpKey: mcpKey, admin: admin,
			target:     "/mcp?key=" + mcpKey,
			wantStatus: http.StatusOK,
		},
		"Admin secret as bearer (backwards compatible)": {
			mcpKey: mcpKey, admin: admin,
			headers:    map[string]string{"Authorization": "Bearer " + admin},
			wantStatus: http.StatusOK,
		},
		"Admin secret works when no MCP key is configured": {
			admin:      admin,
			headers:    map[string]string{"Authorization": "Bearer " + admin},
			wantStatus: http.StatusOK,
		},
		"Admin secret is not accepted in query": {
			mcpKey: mcpKey, admin: admin,
			target:     "/mcp?key=" + admin,
			wantStatus: http.StatusUnauthorized,
		},
		"Wrong key": {
			mcpKey: mcpKey, admin: admin,
			headers:    map[string]string{"Authorization": "Bearer nope"},
			wantStatus: http.StatusUnauthorized,
		},
		"Missing bearer scheme": {
			mcpKey: mcpKey, admin: admin,
			headers:    map[string]string{"Authorization": mcpKey},
			wantStatus: http.StatusUnauthorized,
		},
		"No credentials": {
			mcpKey: mcpKey, admin: admin,
			wantStatus: http.StatusUnauthorized,
		},
		"Empty bearer cannot match an unset MCP key": {
			admin:      admin,
			headers:    map[string]string{"Authorization": "Bearer "},
			wantStatus: http.StatusUnauthorized,
		},
		"Empty query key cannot match an unset MCP key": {
			admin:      admin,
			target:     "/mcp?key=",
			wantStatus: http.StatusUnauthorized,
		},
		"Fails closed when nothing is configured": {
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			target := test.target
			if target == "" {
				target = "/mcp"
			}
			req := httptest.NewRequest(http.MethodPost, target, nil)
			for k, v := range test.headers {
				req.Header.Set(k, v)
			}
			req = LiftMCPQueryKey(req)

			rec := httptest.NewRecorder()
			nextCalled := false
			next := func(_ *webkit.Context) error {
				nextCalled = true
				return nil
			}

			err := MCPAuth(test.mcpKey, test.admin)(next)(webkit.NewContext(rec, req))

			if test.wantStatus == http.StatusOK {
				require.NoError(t, err)
				assert.True(t, nextCalled)
				return
			}
			assert.False(t, nextCalled)
			var e *webkit.Error
			require.ErrorAs(t, err, &e)
			assert.Equal(t, test.wantStatus, e.Code)
			if test.wantStatus == http.StatusUnauthorized {
				assert.Equal(t, `Bearer realm="godaily"`, rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestLiftMCPQueryKey(t *testing.T) {
	t.Parallel()

	t.Run("Removes key from URL and RequestURI", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/api/mcp/?foo=bar&key=s3cret", nil)
		req.URL.Path = "/mcp" // as after StripPrefix and trailing-slash removal

		got := LiftMCPQueryKey(req)

		assert.Equal(t, "foo=bar", got.URL.RawQuery)
		assert.Equal(t, "/api/mcp/?foo=bar", got.RequestURI)
		assert.NotContains(t, got.URL.String(), "s3cret")
		assert.NotContains(t, got.RequestURI, "s3cret")
		assert.Equal(t, "s3cret", got.Context().Value(mcpQueryKeyCtx{}))
		// The original request is left untouched.
		assert.Contains(t, req.URL.RawQuery, "s3cret")
	})

	t.Run("Only key in query", func(t *testing.T) {
		t.Parallel()

		got := LiftMCPQueryKey(httptest.NewRequest(http.MethodPost, "/api/mcp?key=s3cret", nil))
		assert.Empty(t, got.URL.RawQuery)
		assert.Equal(t, "/api/mcp", got.RequestURI)
	})

	t.Run("No key leaves request unchanged", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/api/mcp?foo=bar", nil)
		assert.Same(t, req, LiftMCPQueryKey(req))
	})
}
