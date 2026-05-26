// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gohttp

import (
	"crypto/rand"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// retryTransport wraps an inner http.RoundTripper and retries eligible
// requests with exponential back-off and full jitter.
//
// Only methods listed in opts.retryMethods are ever retried. GET and HEAD
// carry no body, so body rewinding is not needed. If future callers need
// POST retries they must ensure the body can be re-read themselves.
type retryTransport struct {
	inner http.RoundTripper
	opts  *options
}

// RoundTrip implements http.RoundTripper.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.opts.retryMethods[req.Method] {
		return t.inner.RoundTrip(req)
	}

	var (
		resp *http.Response
		err  error
	)

	for attempt := range t.opts.maxAttempts {
		resp, err = t.inner.RoundTrip(req)

		isLast := attempt == t.opts.maxAttempts-1

		if err == nil && !t.opts.retryStatuses[resp.StatusCode] {
			return resp, nil
		}
		if isLast {
			return resp, err
		}

		// Drain and discard the body so the underlying connection can be reused.
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		delay := jitteredDelay(attempt, t.opts.baseDelay, t.opts.maxDelay)

		// Respect Retry-After on 429 responses, capped at maxDelay.
		if err == nil && resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			if d := parseRetryAfter(resp.Header.Get("Retry-After")); d > delay {
				delay = min(d, t.opts.maxDelay)
			}
		}

		select {
		case <-time.After(delay):
		case <-req.Context().Done():
			return nil, errors.Wrap(req.Context().Err(), "gohttp: context cancelled during retry backoff")
		}
	}

	// Unreachable — loop always returns on last attempt.
	return resp, err
}

// jitteredDelay returns a random duration in [0, cap) where cap is
// min(maxDelay, baseDelay*2^attempt). This "full jitter" strategy avoids
// thundering-herd on simultaneous retries.
func jitteredDelay(attempt int, base, max time.Duration) time.Duration {
	cap := base * (1 << attempt)
	if cap > max || cap <= 0 {
		cap = max
	}
	if cap <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(cap)))
	if err != nil {
		return cap / 2
	}
	return time.Duration(n.Int64())
}

// parseRetryAfter parses the Retry-After header value as integer seconds.
// HTTP-date format is not supported; on any parse failure zero is returned.
func parseRetryAfter(s string) time.Duration {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}
