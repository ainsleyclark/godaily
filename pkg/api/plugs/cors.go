// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugs

import (
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

// CORS is a webkit.Plug that opens the API up to cross-origin browser
// requests and short-circuits OPTIONS preflight with 204.
//
// Wildcard origin is safe here because every authenticated route gates on a
// Bearer token (see Auth) and we never use cookies — a hostile browser can
// preflight but can't read anything without the secret. Non-browser callers
// (curl, scripts, MCP tools) ignore CORS entirely.
func CORS(next webkit.Handler) webkit.Handler {
	return func(c *webkit.Context) error {
		h := c.Response.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		h.Set("Access-Control-Max-Age", "600")
		if c.Request.Method == http.MethodOptions {
			c.Response.WriteHeader(http.StatusNoContent)
			return nil
		}
		return next(c)
	}
}
