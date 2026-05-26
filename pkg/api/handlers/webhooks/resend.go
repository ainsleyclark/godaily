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

package webhooks

import (
	"io"
	"log/slog"
	"net/http"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	svcengagement "github.com/ainsleyclark/godaily/pkg/services/engagement"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Handler holds the narrow dependencies for webhook HTTP handlers.
type Handler struct {
	emailEvents *svcengagement.EventService
	config      *env.Config
}

// New constructs a Handler from the application App.
func New(a *godaily.App) *Handler {
	return &Handler{
		emailEvents: a.EmailEvents,
		config:      a.Config,
	}
}

// Routes registers all webhook routes on kit.
func (h *Handler) Routes(kit *webkit.Kit) {
	kit.Post("/resend", h.Resend)
}

// Resend handles POST /webhooks/resend.
// The endpoint is public but every request is verified against the Svix-style
// signature Resend includes; unsigned or tampered requests are rejected.
func (h *Handler) Resend(c *webkit.Context) error {
	ctx := c.Context()

	secret := h.config.ResendWebhookSecret
	if secret == "" {
		return webkit.NewError(http.StatusInternalServerError, "resend webhook secret is not configured")
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return webkit.NewError(http.StatusBadRequest, "cannot read request body")
	}

	headers := email.WebhookHeaders{
		ID:        c.Request.Header.Get("svix-id"),
		Timestamp: c.Request.Header.Get("svix-timestamp"),
		Signature: c.Request.Header.Get("svix-signature"),
	}
	if err = email.VerifyWebhook(string(body), headers, secret); err != nil {
		slog.WarnContext(ctx, "Rejected Resend webhook with invalid signature", "err", err)
		return webkit.NewError(http.StatusUnauthorized, "invalid signature")
	}

	evt, err := email.ParseWebhook(body)
	if err != nil {
		return webkit.NewError(http.StatusBadRequest, "invalid payload")
	}

	domainEvt, tracked, err := email.ToEmailEvent(evt, headers.ID)
	if err != nil {
		return webkit.NewError(http.StatusBadRequest, "invalid payload")
	}
	if !tracked {
		slog.WarnContext(ctx, "Rejected Resend webhook with invalid event", "evt", evt)
		return c.NoContent(http.StatusOK)
	}

	if err = h.emailEvents.Process(ctx, domainEvt); err != nil {
		slog.ErrorContext(ctx, "Failed to process Resend webhook event", "type", domainEvt.Type, "err", err)
		return webkit.NewError(http.StatusInternalServerError, "failed to process event")
	}

	return c.NoContent(http.StatusOK)
}
