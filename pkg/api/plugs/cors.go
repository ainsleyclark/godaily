// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugs

import "net/http"

// CORS returns an HTTP middleware that opens the API up to cross-origin browser
// requests and short-circuits OPTIONS preflight with 204.
//
// Wildcard origin is safe here because every authenticated route gates on a
// Bearer token (see Auth) and we never use cookies — a hostile browser can
// preflight but can't read anything without the secret. Non-browser callers
// (curl, scripts, MCP tools) ignore CORS entirely.
//
// Mounted as chi middleware on the top-level mux (not a webkit.Plug) so OPTIONS
// preflight is answered before chi 405s on routes that only register GET.
func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", "*")
			h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
			h.Set("Access-Control-Max-Age", "600")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
