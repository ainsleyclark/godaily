// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"github.com/ainsleyclark/godaily/pkg"
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
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		social:        a.Social,
		socialPosts:   a.Repository.SocialPosts,
		socialMetrics: a.Repository.SocialMetrics,
		statFetchers:  a.StatFetchers,
		slack:         a.Slack,
		config:        a.Config,
	}
}

// Routes registers all social routes on kit.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Get("/featured", h.Featured, auth)
	kit.Get("/rotation", h.Rotation, auth)
	kit.Get("/metrics", h.Metrics, auth)
}
