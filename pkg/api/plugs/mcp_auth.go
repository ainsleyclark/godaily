// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugs

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

// MCPKeyQueryParam is the query parameter that may carry the MCP key for
// clients that cannot send custom headers (for example claude.ai custom
// connectors without request-header support).
const MCPKeyQueryParam = "key"

// mcpQueryKeyCtx is the context key under which LiftMCPQueryKey stores a
// key taken from the query string.
type mcpQueryKeyCtx struct{}

// LiftMCPQueryKey moves the ?key= query parameter off the request URL and
// into the request context, where MCPAuth reads it. Removing it from the URL
// (and RequestURI) before the request reaches the logger keeps the key out of
// application logs. Requests without the parameter are returned unchanged.
func LiftMCPQueryKey(r *http.Request) *http.Request {
	q := r.URL.Query()
	key := q.Get(MCPKeyQueryParam)
	if !q.Has(MCPKeyQueryParam) {
		return r
	}
	q.Del(MCPKeyQueryParam)

	r2 := r.WithContext(context.WithValue(r.Context(), mcpQueryKeyCtx{}, key))
	u := *r.URL
	u.RawQuery = q.Encode()
	r2.URL = &u

	requestURI, _, _ := strings.Cut(r.RequestURI, "?")
	if u.RawQuery != "" {
		requestURI += "?" + u.RawQuery
	}
	r2.RequestURI = requestURI
	return r2
}

// MCPAuth returns a webkit.Plug guarding the MCP endpoint.
//
// A request is let through when it presents either key:
//   - mcpKey, the read-only key handed to MCP users, via an
//     "Authorization: Bearer <key>" header, an "X-API-Key: <key>" header, or
//     the ?key= query parameter (see LiftMCPQueryKey).
//   - adminSecret (API_SECRET), via either header, so existing admin
//     credentials keep working. It is deliberately not accepted in the query
//     string, where it could leak into URLs and proxy logs.
//
// Unlike Auth, MCPAuth fails closed: when neither key is configured every
// request is rejected with 503 rather than allowed through.
func MCPAuth(mcpKey, adminSecret string) webkit.Plug {
	return func(next webkit.Handler) webkit.Handler {
		return func(c *webkit.Context) error {
			if mcpKey == "" && adminSecret == "" {
				return webkit.NewError(http.StatusServiceUnavailable, "MCP server is not configured")
			}
			if !MCPAuthenticated(c.Request, mcpKey, adminSecret) {
				c.Response.Header().Set("WWW-Authenticate", `Bearer realm="godaily"`)
				return webkit.NewError(http.StatusUnauthorized, "unauthorized")
			}
			return next(c)
		}
	}
}

// MCPAuthenticated reports whether r carries a credential MCPAuth accepts.
// It is also used to exempt MCP clients from the public rate limit: claude.ai
// and other hosted clients call from shared egress IPs, so a per-IP limit
// would throttle every user of them together.
func MCPAuthenticated(r *http.Request, mcpKey, adminSecret string) bool {
	if key, ok := r.Context().Value(mcpQueryKeyCtx{}).(string); ok && secretEqual(key, mcpKey) {
		return true
	}
	for _, presented := range headerKeys(r) {
		if secretEqual(presented, mcpKey) || secretEqual(presented, adminSecret) {
			return true
		}
	}
	return false
}

// headerKeys returns the credentials presented in the Authorization (Bearer
// scheme) and X-API-Key headers.
func headerKeys(r *http.Request) []string {
	var keys []string
	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok && token != "" {
		keys = append(keys, token)
	}
	if key := r.Header.Get("X-API-Key"); key != "" {
		keys = append(keys, key)
	}
	return keys
}

// secretEqual compares a presented credential with a configured secret in
// constant time. An empty secret never matches, so an unset key cannot be
// satisfied by an empty credential.
func secretEqual(presented, secret string) bool {
	if secret == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented), []byte(secret)) == 1
}
