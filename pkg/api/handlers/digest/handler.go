// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	digestdomain "github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler holds the narrow dependencies for digest HTTP handlers.
type Handler struct {
	runner          digestdomain.Service
	subscribers     audience.SubscriberService
	subscribersRepo audience.SubscriberRepository
	issuesRepo      digestdomain.IssueRepository
	slack           slack.Sender
	config          *env.Config
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		runner:          a.Service.Digest,
		subscribers:     a.Service.Subscribers,
		subscribersRepo: a.Repository.Subscribers,
		issuesRepo:      a.Repository.Issues,
		slack:           a.Slack,
		config:          a.Config,
	}
}

// Routes registers the authenticated digest pipeline routes on kit.
// Public subscriber lifecycle routes (subscribe, confirm, unsubscribe) are
// registered at the root level in the mux, not here.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Get("/collect", h.Collect, auth)
	kit.Get("/build", h.Build, auth)
	kit.Get("/send", h.Send, auth)
	kit.Get("/nudge", h.Nudge, auth)
	kit.Get("/issues", h.Issues, auth)
	kit.Get("/subscribers", h.Subscribers, auth)
}
