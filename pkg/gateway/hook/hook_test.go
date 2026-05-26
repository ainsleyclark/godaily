// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hook

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeartbeat(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		useServer  bool
		wantMethod string
	}{
		"No-op on empty URL": {},
		"Fires GET": {
			useServer:  true,
			wantMethod: http.MethodGet,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var gotMethod string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
			}))
			defer srv.Close()

			url := ""
			if test.useServer {
				url = srv.URL
			}

			Heartbeat(t.Context(), url)
			assert.Equal(t, test.wantMethod, gotMethod)
		})
	}
}

func TestDeploy(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		useServer  bool
		wantMethod string
	}{
		"No-op on empty URL": {},
		"Fires POST": {
			useServer:  true,
			wantMethod: http.MethodPost,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var gotMethod string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
			}))
			defer srv.Close()

			url := ""
			if test.useServer {
				url = srv.URL
			}

			Deploy(t.Context(), url)
			assert.Equal(t, test.wantMethod, gotMethod)
		})
	}
}
