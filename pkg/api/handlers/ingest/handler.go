// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ingest holds HTTP handlers for manually ingesting source data when a
// source's live fetch is blocked (e.g. Reddit via ScraperAPI). Each route
// accepts a source's raw payload, transforms it through the standard pipeline,
// and persists the items for the current collection window.
package ingest

import (
	godaily "github.com/ainsleyclark/godaily/pkg"
	digestdomain "github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler holds the narrow dependencies for ingest HTTP handlers.
type Handler struct {
	runner digestdomain.Service
	slack  slack.Sender
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		runner: a.Service.Digest,
		slack:  a.Slack,
	}
}

// Routes registers the authenticated ingest routes on kit.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Post("/reddit", h.Reddit, auth)
}
