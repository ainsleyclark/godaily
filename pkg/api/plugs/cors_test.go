// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	t.Parallel()

	const allowed = "https://analytics.godaily.dev"

	tt := map[string]struct {
		origins         []string
		method          string
		origin          string
		wantStatus      int
		wantAllowOrigin string
		wantNextCalled  bool
	}{
		"Matching origin on GET sets headers and calls next": {
			origins:         []string{allowed},
			method:          http.MethodGet,
			origin:          allowed,
			wantStatus:      http.StatusOK,
			wantAllowOrigin: allowed,
			wantNextCalled:  true,
		},
		"Non-matching origin on GET omits headers and calls next": {
			origins:         []string{allowed},
			method:          http.MethodGet,
			origin:          "https://evil.example.com",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "",
			wantNextCalled:  true,
		},
		"GET with no origin header passes through unchanged": {
			origins:         []string{allowed},
			method:          http.MethodGet,
			origin:          "",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "",
			wantNextCalled:  true,
		},
		"OPTIONS preflight from matching origin short-circuits with 204 and headers": {
			origins:         []string{allowed},
			method:          http.MethodOptions,
			origin:          allowed,
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: allowed,
			wantNextCalled:  false,
		},
		"OPTIONS preflight from non-matching origin still 204 but no headers": {
			origins:         []string{allowed},
			method:          http.MethodOptions,
			origin:          "https://evil.example.com",
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: "",
			wantNextCalled:  false,
		},
		"Empty allow-list disables CORS entirely": {
			origins:         nil,
			method:          http.MethodGet,
			origin:          allowed,
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "",
			wantNextCalled:  true,
		},
		"Whitespace in origins is trimmed before matching": {
			origins:         []string{"  " + allowed + "  "},
			method:          http.MethodGet,
			origin:          allowed,
			wantStatus:      http.StatusOK,
			wantAllowOrigin: allowed,
			wantNextCalled:  true,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := CORS(test.origins)(next)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(test.method, "/", nil)
			if test.origin != "" {
				r.Header.Set("Origin", test.origin)
			}
			handler.ServeHTTP(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
			assert.Equal(t, test.wantNextCalled, nextCalled)
			assert.Equal(t, test.wantAllowOrigin, w.Header().Get("Access-Control-Allow-Origin"))

			if test.wantAllowOrigin != "" {
				assert.Contains(t, w.Header().Values("Vary"), "Origin")
				assert.Equal(t, "GET, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Authorization, Content-Type, Accept", w.Header().Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "600", w.Header().Get("Access-Control-Max-Age"))
			} else {
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Methods"))
			}
		})
	}
}
