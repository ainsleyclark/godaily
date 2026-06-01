// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

import (
	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler holds the dependencies for metrics HTTP handlers.
type Handler struct {
	metricsRepo    engagement.MetricsRepository
	issuesRepo     digest.IssueRepository
	emailEvents    engagement.EmailEventRepository
	metricsService engagement.MetricsService
	socialMetrics  engagement.SocialMetricRepository
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		metricsRepo:    a.Repository.Metrics,
		issuesRepo:     a.Repository.Issues,
		emailEvents:    a.Repository.EmailEvents,
		metricsService: a.Service.Metrics,
		socialMetrics:  a.Repository.SocialMetrics,
	}
}

// Routes registers all metrics routes on kit.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Get("/summary", h.Summary, auth)
	kit.Get("/issues", h.Issues, auth)
	kit.Get("/issues/{slug}", h.IssueBySlug, auth)
	kit.Get("/issues/{slug}/trend", h.IssueTrend, auth)
	kit.Get("/items", h.Items, auth)
	kit.Get("/tags", h.Tags, auth)
	kit.Get("/sources", h.Sources, auth)
	kit.Get("/trend", h.Trend, auth)
	kit.Get("/subscribers", h.Subscribers, auth)
	kit.Get("/roundup", h.Roundup, auth)
	kit.Get("/social", h.Social, auth)
}
