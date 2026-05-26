// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugs

import (
	"net/http"
	"strings"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Auth returns a webkit.Plug that requires a valid Bearer token in the
// Authorization header. When secret is empty the plug is a no-op, preserving
// dev/CI behaviour.
func Auth(secret string) webkit.Plug {
	return func(next webkit.Handler) webkit.Handler {
		return func(c *webkit.Context) error {
			if !authenticated(c.Request, secret) {
				return webkit.NewError(http.StatusUnauthorized, "unauthorized")
			}
			return next(c)
		}
	}
}

// authenticated checks that the request carries the expected secret in the
// Authorization header (Bearer scheme). Returns true unconditionally when
// secret is empty so that local development and CI work without credentials.
func authenticated(r *http.Request, secret string) bool {
	if secret == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(auth, "Bearer ")
	return ok && token == secret
}
