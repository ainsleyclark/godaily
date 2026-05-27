// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugs

import (
	"net/http"
	"strings"
)

// CORS returns an HTTP middleware that adds CORS headers for matching origins
// and short-circuits OPTIONS preflight requests with 204.
//
// It's a plain net/http middleware (not a webkit.Plug) so it can be mounted on
// a chi sub-router via Mux().Use(...). That placement is required: preflight
// requests target methods that aren't registered on the route (only GET is),
// so chi would 405 before any per-route webkit.Plug ran. Mounted on the
// sub-router, the middleware runs first and can answer the preflight directly.
//
// Origins are matched exact-string; the wildcard "*" is not supported because
// the dashboard sends an Authorization header and credentialed CORS doesn't
// allow wildcards. Pass an empty slice to disable CORS entirely.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = struct{}{}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Add("Vary", "Origin")
				h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
				h.Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
