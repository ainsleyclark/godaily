// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/api/handlers/social/drafts"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler holds the dependencies for social HTTP handlers.
type Handler struct {
	social        social.Service
	socialPosts   social.PostRepository
	socialMetrics engagement.SocialMetricRepository
	statFetchers  map[social.Platform]platform.StatFetcher
	slack         slack.Sender
	config        *env.Config
	drafts        *drafts.Handler
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		social:        a.Service.Social,
		socialPosts:   a.Repository.SocialPosts,
		socialMetrics: a.Repository.SocialMetrics,
		statFetchers:  a.StatFetchers,
		slack:         a.Slack,
		config:        a.Config,
		drafts:        drafts.New(a),
	}
}

// Routes registers all /social routes on kit. /social/drafts is
// delegated to the drafts subpackage so its richer lifecycle (list /
// edit / cancel + per-row status checks) stays self-contained.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Get("/publish", h.Publish, auth)
	kit.Get("/metrics", h.Metrics, auth)
	kit.Group("/drafts", func(k *webkit.Kit) { h.drafts.Routes(k, auth) })
}
