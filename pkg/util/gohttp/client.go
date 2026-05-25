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

// Package gohttp provides an *http.Client factory with automatic retry and
// exponential back-off. The retry is implemented as an http.RoundTripper
// middleware so callers use the client exactly like a plain *http.Client.
//
// Only idempotent methods (GET, HEAD by default) are retried. POST and other
// non-idempotent methods bypass retry entirely to avoid duplicate side effects.
//
// The Timeout field on the returned client is end-to-end across all attempts.
// For per-attempt deadlines, callers should pass a context.WithTimeout to each
// request.
package gohttp

import (
	"net/http"
	"time"
)

type (
	// Option configures the retry behaviour of a client returned by New.
	Option func(*options)

	options struct {
		timeout       time.Duration
		maxAttempts   int
		baseDelay     time.Duration
		maxDelay      time.Duration
		retryMethods  map[string]bool
		retryStatuses map[int]bool
	}
)

func defaultOptions() options {
	return options{
		timeout:     30 * time.Second,
		maxAttempts: 3,
		baseDelay:   200 * time.Millisecond,
		maxDelay:    10 * time.Second,
		retryMethods: map[string]bool{
			http.MethodGet:  true,
			http.MethodHead: true,
		},
		retryStatuses: map[int]bool{
			http.StatusTooManyRequests:     true,
			http.StatusInternalServerError: true,
			http.StatusBadGateway:          true,
			http.StatusServiceUnavailable:  true,
			http.StatusGatewayTimeout:      true,
		},
	}
}

// WithTimeout sets the end-to-end timeout on the returned *http.Client.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// WithMaxAttempts sets the maximum number of attempts, including the first.
// Must be at least 1.
func WithMaxAttempts(n int) Option {
	return func(o *options) { o.maxAttempts = n }
}

// WithBaseDelay sets the base delay for exponential back-off.
func WithBaseDelay(d time.Duration) Option {
	return func(o *options) { o.baseDelay = d }
}

// WithMaxDelay sets the ceiling for computed back-off delays.
func WithMaxDelay(d time.Duration) Option {
	return func(o *options) { o.maxDelay = d }
}

// WithRetryMethods replaces the default set of HTTP methods that are eligible
// for retry. Pass method strings as defined in net/http (e.g. http.MethodGet).
func WithRetryMethods(methods ...string) Option {
	return func(o *options) {
		m := make(map[string]bool, len(methods))
		for _, method := range methods {
			m[method] = true
		}
		o.retryMethods = m
	}
}

// WithRetryStatuses replaces the default set of HTTP status codes that trigger
// a retry.
func WithRetryStatuses(codes ...int) Option {
	return func(o *options) {
		s := make(map[int]bool, len(codes))
		for _, code := range codes {
			s[code] = true
		}
		o.retryStatuses = s
	}
}

// New returns an *http.Client with retry and exponential back-off configured
// via a RoundTripper middleware. Defaults: 3 attempts, 200 ms base delay,
// 10 s max delay, 30 s client timeout, retry on GET/HEAD, retry statuses
// 429/500/502/503/504.
func New(opts ...Option) *http.Client {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxAttempts < 1 {
		o.maxAttempts = 1
	}
	return &http.Client{
		Timeout: o.timeout,
		Transport: &retryTransport{
			inner: http.DefaultTransport,
			opts:  &o,
		},
	}
}
