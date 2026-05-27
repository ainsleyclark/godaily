// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugs

import (
	"net"
	"net/http"
	"sync"

	"github.com/ainsleydev/webkit/pkg/webkit"
	"golang.org/x/time/rate"
)

// RateLimit returns a webkit.Plug that enforces per-IP rate limiting using the
// provided RateLimiter.
func RateLimit(limiter *RateLimiter) webkit.Plug {
	return func(next webkit.Handler) webkit.Handler {
		return func(c *webkit.Context) error {
			if !limiter.Allow(ClientIP(c.Request)) {
				return webkit.NewError(http.StatusTooManyRequests, "rate limit exceeded")
			}
			return next(c)
		}
	}
}

// Limiter is the shared rate limiter for public API endpoints.
// Allows 1 request per second with a burst of 10 per unique client IP.
var Limiter = NewRateLimiter(1, 50)

// RateLimiter holds a per-IP token-bucket limiter map.
// Implementations must be used via NewRateLimiter.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// NewRateLimiter creates a RateLimiter that allows rps requests per second
// with the given burst size per client IP.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// Allow reports whether the client identified by ip is within the rate limit.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	l, ok := rl.limiters[ip]
	if !ok {
		l = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[ip] = l
	}
	rl.mu.Unlock()
	return l.Allow()
}

// ClientIP extracts the client IP from X-Forwarded-For (first entry) or
// RemoteAddr, stripping the port if present.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := len(xff); idx > 0 {
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					idx = i
					break
				}
			}
			return xff[:idx]
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
