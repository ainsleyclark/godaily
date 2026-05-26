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

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Limit(t *testing.T) {
	t.Parallel()

	t.Run("Passes through within limit", func(t *testing.T) {
		t.Parallel()

		rl := NewRateLimiter(10, 10)
		handler := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()

		handler(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Blocks when limit exceeded", func(t *testing.T) {
		t.Parallel()

		// Burst of 1 means the second request from the same IP is rejected.
		rl := NewRateLimiter(0.0001, 1)
		handler := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "5.6.7.8:5678"

		// First request consumes the single token.
		w1 := httptest.NewRecorder()
		handler(w1, r)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request is rejected immediately (no refill at this rate).
		w2 := httptest.NewRecorder()
		handler(w2, r)
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	})

	t.Run("Different IPs are tracked independently", func(t *testing.T) {
		t.Parallel()

		rl := NewRateLimiter(0.0001, 1)
		handler := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		makeReq := func(ip string) int {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = ip + ":1234"
			w := httptest.NewRecorder()
			handler(w, r)
			return w.Code
		}

		// Each fresh IP gets its own bucket.
		assert.Equal(t, http.StatusOK, makeReq("10.0.0.1"))
		assert.Equal(t, http.StatusOK, makeReq("10.0.0.2"))
	})

	t.Run("X-Forwarded-For header used for IP", func(t *testing.T) {
		t.Parallel()

		rl := NewRateLimiter(0.0001, 1)
		handler := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "127.0.0.1:9999"
		r.Header.Set("X-Forwarded-For", "203.0.113.5, 192.168.1.1")

		w1 := httptest.NewRecorder()
		handler(w1, r)
		assert.Equal(t, http.StatusOK, w1.Code)

		w2 := httptest.NewRecorder()
		handler(w2, r)
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
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

			got := ClientIP(r)
			assert.Equal(t, test.want, got)
		})
	}
}
