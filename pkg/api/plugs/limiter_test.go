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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var passHandler webkit.Handler = func(_ *webkit.Context) error { return nil }

func newCtx(ip string) *webkit.Context {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = ip + ":1234"
	return webkit.NewContext(httptest.NewRecorder(), r)
}

func TestRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("Passes through within limit", func(t *testing.T) {
		t.Parallel()
		plug := RateLimit(NewRateLimiter(10, 10))
		err := plug(passHandler)(newCtx("1.2.3.4"))
		require.NoError(t, err)
	})

	t.Run("Blocks when limit exceeded", func(t *testing.T) {
		t.Parallel()
		handler := RateLimit(NewRateLimiter(0.0001, 1))(passHandler)

		require.NoError(t, handler(newCtx("5.6.7.8")), "first request should pass")

		err := handler(newCtx("5.6.7.8"))
		require.Error(t, err)
		var webErr *webkit.Error
		require.True(t, errors.As(err, &webErr))
		assert.Equal(t, http.StatusTooManyRequests, webErr.Code)
	})

	t.Run("Different IPs are tracked independently", func(t *testing.T) {
		t.Parallel()
		handler := RateLimit(NewRateLimiter(0.0001, 1))(passHandler)

		require.NoError(t, handler(newCtx("10.0.0.1")))
		require.NoError(t, handler(newCtx("10.0.0.2")))
	})

	t.Run("X-Forwarded-For header used for IP", func(t *testing.T) {
		t.Parallel()
		handler := RateLimit(NewRateLimiter(0.0001, 1))(passHandler)

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "127.0.0.1:9999"
		r.Header.Set("X-Forwarded-For", "203.0.113.5, 192.168.1.1")

		require.NoError(t, handler(webkit.NewContext(httptest.NewRecorder(), r)))

		err := handler(webkit.NewContext(httptest.NewRecorder(), r))
		require.Error(t, err)
		var webErr *webkit.Error
		require.True(t, errors.As(err, &webErr))
		assert.Equal(t, http.StatusTooManyRequests, webErr.Code)
	})
}

func TestClientIP(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		remoteAddr string
		xff        string
		want       string
	}{
		"RemoteAddr with port":         {remoteAddr: "1.2.3.4:5678", want: "1.2.3.4"},
		"RemoteAddr without port":      {remoteAddr: "1.2.3.4", want: "1.2.3.4"},
		"X-Forwarded-For single":       {remoteAddr: "127.0.0.1:80", xff: "9.9.9.9", want: "9.9.9.9"},
		"X-Forwarded-For with proxies": {remoteAddr: "127.0.0.1:80", xff: "9.9.9.9, 10.0.0.1", want: "9.9.9.9"},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = test.remoteAddr
			if test.xff != "" {
				r.Header.Set("X-Forwarded-For", test.xff)
			}

			assert.Equal(t, test.want, ClientIP(r))
		})
	}
}
