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
		runner:          a.Runner,
		subscribers:     a.Subscribers,
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
	kit.Get("/preview", h.Preview, auth)
	kit.Get("/issues", h.Issues, auth)
	kit.Get("/subscribers", h.Subscribers, auth)
}
