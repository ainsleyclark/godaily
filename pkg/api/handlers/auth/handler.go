// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler holds the dependencies for auth HTTP handlers.
type Handler struct {
	password  string
	apiSecret string
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		password:  a.Config.DashboardPassword,
		apiSecret: a.Config.APISecret,
	}
}

// Routes registers all auth routes on kit. Login is intentionally
// unauthenticated so the dashboard can exchange a password for a token.
func (h *Handler) Routes(kit *webkit.Kit) {
	kit.Post("/login", h.Login)
}
