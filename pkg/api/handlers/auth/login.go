// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"crypto/subtle"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Login handles POST /auth/login.
// It exchanges the dashboard password for the API secret, which the dashboard
// then uses as a Bearer token. When no password is configured (dev/CI) any
// request succeeds, mirroring the no-op behaviour of the auth plug.
func (h *Handler) Login(c *webkit.Context) error {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.BindJSON(&body); err != nil {
		return webkit.NewError(http.StatusBadRequest, "password is required")
	}

	if h.password != "" &&
		subtle.ConstantTimeCompare([]byte(body.Password), []byte(h.password)) != 1 {
		return webkit.NewError(http.StatusUnauthorized, "invalid password")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]string{"token": h.apiSecret},
	})
}
