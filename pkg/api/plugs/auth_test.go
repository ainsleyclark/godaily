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
