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

package gohttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		opts        []Option
		wantTimeout time.Duration
	}{
		"Default timeout": {
			opts:        nil,
			wantTimeout: 30 * time.Second,
		},
		"Custom timeout": {
			opts:        []Option{WithTimeout(5 * time.Second)},
			wantTimeout: 5 * time.Second,
		},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := New(test.opts...)
			require.NotNil(t, c)
			assert.Equal(t, test.wantTimeout, c.Timeout)
			assert.IsType(t, &retryTransport{}, c.Transport)
		})
	}
}

func TestNew_MaxAttemptsFloor(t *testing.T) {
	t.Parallel()
	c := New(WithMaxAttempts(0))
	rt := c.Transport.(*retryTransport)
	assert.Equal(t, 1, rt.opts.maxAttempts)
}

func TestRetryTransport_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("Success on first attempt", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := newFastClient()
		resp, err := c.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("Retries on 503 then succeeds", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			n := calls.Add(1)
			if n < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := newFastClient()
		resp, err := c.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, int32(3), calls.Load())
	})

	t.Run("Exhausts retries returns last response", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		c := newFastClient()
		resp, err := c.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assert.Equal(t, int32(3), calls.Load())
	})

	t.Run("POST not retried on 503", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		c := newFastClient()
		resp, err := c.Post(srv.URL, "application/json", http.NoBody) //nolint:noctx
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("Non-retryable status not retried", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c := newFastClient()
		resp, err := c.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("Context cancellation aborts retry wait", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		// Long delay so the retry sleep would block without context cancellation.
		c := New(
			WithMaxAttempts(3),
			WithBaseDelay(10*time.Second),
			WithMaxDelay(30*time.Second),
		)

		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		require.NoError(t, err)

		start := time.Now()
		_, err = c.Do(req)
		elapsed := time.Since(start)

		assert.Error(t, err)
		// Should cancel well before the 10s base delay.
		assert.Less(t, elapsed, 2*time.Second)
	})

	t.Run("Network error retried", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32

		// Use a handler that counts calls; close the server after the first to
		// simulate a network error on subsequent attempts by using a proxy approach.
		// Simpler: use a counter and close the connection for the first N attempts.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := calls.Add(1)
			if n == 1 {
				// Hijack and close the connection to simulate a network error.
				hj, ok := w.(http.Hijacker)
				if !ok {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				conn, _, _ := hj.Hijack()
				conn.Close()
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := newFastClient()
		resp, err := c.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, int32(2), calls.Load())
	})

	t.Run("429 with Retry-After header respected", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			n := calls.Add(1)
			if n == 1 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := New(
			WithMaxAttempts(3),
			WithBaseDelay(0),
			WithMaxDelay(2*time.Second),
		)

		start := time.Now()
		resp, err := c.Get(srv.URL)
		elapsed := time.Since(start)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, int32(2), calls.Load())
		// Should have waited at least ~1s for the Retry-After.
		assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond)
	})
}

func TestJitteredDelay(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		attempt int
		base    time.Duration
		max     time.Duration
	}{
		"Attempt 0": {attempt: 0, base: 100 * time.Millisecond, max: 10 * time.Second},
		"Attempt 5": {attempt: 5, base: 100 * time.Millisecond, max: 10 * time.Second},
		"Zero base": {attempt: 2, base: 0, max: 10 * time.Second},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for range 100 {
				got := jitteredDelay(test.attempt, test.base, test.max)
				assert.GreaterOrEqual(t, got, time.Duration(0))
				assert.LessOrEqual(t, got, test.max)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	tt := map[string]struct {
		input string
		want  time.Duration
	}{
		"Empty string":    {input: "", want: 0},
		"Valid seconds":   {input: "5", want: 5 * time.Second},
		"Zero":            {input: "0", want: 0},
		"Negative":        {input: "-1", want: 0},
		"Non-numeric":     {input: "abc", want: 0},
		"HTTP-date":       {input: "Wed, 21 Oct 2015 07:28:00 GMT", want: 0},
	}

	for name, test := range tt {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := parseRetryAfter(test.input)
			assert.Equal(t, test.want, got)
		})
	}
}

// newFastClient returns a retry client with near-zero delays, suitable for tests.
func newFastClient() *http.Client {
	return New(
		WithMaxAttempts(3),
		WithBaseDelay(time.Millisecond),
		WithMaxDelay(5*time.Millisecond),
	)
}
