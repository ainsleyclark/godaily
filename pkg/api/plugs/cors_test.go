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

	tt := map[string]struct {
		method         string
		origin         string
		wantStatus     int
		wantNextCalled bool
	}{
		"GET passes through with CORS headers": {
			method:         http.MethodGet,
			origin:         "https://analytics.godaily.dev",
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		"GET from any origin still gets headers": {
			method:         http.MethodGet,
			origin:         "https://random-tool.example.com",
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		"GET with no origin (curl / server-to-server) still passes": {
			method:         http.MethodGet,
			origin:         "",
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		"OPTIONS preflight short-circuits with 204": {
			method:         http.MethodOptions,
			origin:         "https://analytics.godaily.dev",
			wantStatus:     http.StatusNoContent,
			wantNextCalled: false,
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

			handler := CORS()(next)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(test.method, "/", nil)
			if test.origin != "" {
				r.Header.Set("Origin", test.origin)
			}
			handler.ServeHTTP(w, r)

			assert.Equal(t, test.wantStatus, w.Code)
			assert.Equal(t, test.wantNextCalled, nextCalled)
			assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "GET, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, "Authorization, Content-Type, Accept", w.Header().Get("Access-Control-Allow-Headers"))
			assert.Equal(t, "600", w.Header().Get("Access-Control-Max-Age"))
		})
	}
}
