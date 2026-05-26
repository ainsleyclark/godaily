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

func TestAuth(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		secret     string
		header     string
		wantStatus int
		wantNext   bool
	}{
		"No secret allows request through": {
			secret:     "",
			header:     "",
			wantStatus: http.StatusOK,
			wantNext:   true,
		},
		"Valid token calls next handler": {
			secret:     "supersecret",
			header:     "Bearer supersecret",
			wantStatus: http.StatusOK,
			wantNext:   true,
		},
		"Invalid token returns 401": {
			secret:     "supersecret",
			header:     "Bearer wrong",
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
		},
		"Missing header returns 401": {
			secret:     "supersecret",
			header:     "",
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false
			next := webkit.Handler(func(_ *webkit.Context) error {
				nextCalled = true
				return nil
			})

			handler := Auth(test.secret)(next)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.header != "" {
				r.Header.Set("Authorization", test.header)
			}

			ctx := webkit.NewContext(w, r)
			err := handler(ctx)

			assert.Equal(t, test.wantNext, nextCalled)
			if !test.wantNext {
				require.Error(t, err)
				var httpErr *webkit.Error
				require.ErrorAs(t, err, &httpErr)
				assert.Equal(t, test.wantStatus, httpErr.Code)
			}
		})
	}
}

func TestAuthenticated(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		secret string
		header string
		want   bool
	}{
		"No secret configured allows any request": {
			secret: "",
			header: "",
			want:   true,
		},
		"No secret configured allows request with header": {
			secret: "",
			header: "Bearer anything",
			want:   true,
		},
		"Correct bearer token": {
			secret: "supersecret",
			header: "Bearer supersecret",
			want:   true,
		},
		"Wrong token": {
			secret: "supersecret",
			header: "Bearer wrongtoken",
			want:   false,
		},
		"Missing header": {
			secret: "supersecret",
			header: "",
			want:   false,
		},
		"Token without Bearer prefix": {
			secret: "supersecret",
			header: "supersecret",
			want:   false,
		},
		"Empty token with secret set": {
			secret: "supersecret",
			header: "Bearer ",
			want:   false,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.header != "" {
				r.Header.Set("Authorization", test.header)
			}
			assert.Equal(t, test.want, authenticated(r, test.secret))
		})
	}
}
