// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webhooks

import (
	"github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler the narrow dependencies for webhook HTTP handlers.
type Handler struct {
	emailEvents engagement.EventService
	config      *env.Config
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		emailEvents: a.Service.Events,
		config:      a.Config,
	}
}

// Routes registers all webhook routes on kit.
func (h *Handler) Routes(kit *webkit.Kit) {
	kit.Post("/resend", h.Resend)
}
