// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package metrics

import (
	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler holds the narrow dependencies for metrics HTTP handlers.
type Handler struct {
	metricsRepo     engagement.MetricsRepository
	issuesRepo      digest.IssueRepository
	emailEvents     engagement.EmailEventRepository
	metricsReporter engagement.MetricsReporter
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		metricsRepo:     a.Repository.Metrics,
		issuesRepo:      a.Repository.Issues,
		emailEvents:     a.Repository.EmailEvents,
		metricsReporter: a.MetricsService,
	}
}

// Routes registers all metrics routes on kit.
func (h *Handler) Routes(kit *webkit.Kit, auth webkit.Plug) {
	kit.Get("/summary", h.Summary, auth)
	kit.Get("/issues", h.Issues, auth)
	kit.Get("/issues/{slug}", h.IssueBySlug, auth)
	kit.Get("/items", h.Items, auth)
	kit.Get("/tags", h.Tags, auth)
	kit.Get("/sources", h.Sources, auth)
	kit.Get("/trend", h.Trend, auth)
	kit.Get("/subscribers", h.Subscribers, auth)
	kit.Get("/roundup", h.Roundup, auth)
}
