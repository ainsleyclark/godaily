// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webhooks

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
)

// Resend godoc
//
//	@Summary		Resend email webhook.
//	@Description	Receives Resend email events (opens, clicks, bounces). Public, but every request is verified against its Svix-style signature; unsigned or tampered requests are rejected.
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			svix-id			header		string			true	"Svix message ID"
//	@Param			svix-timestamp	header		string			true	"Svix timestamp"
//	@Param			svix-signature	header		string			true	"Svix signature"
//	@Success		200				{object}	api.MessageResponse	"Successfully tracked event"
//	@Failure		400				{object}	api.MessageResponse	"Misconfigured, unreadable, or invalid payload"
//	@Failure		401				{object}	api.MessageResponse	"Invalid signature"
//	@Failure		500				{object}	api.MessageResponse	"Failed to process event"
//	@Router			/webhooks/resend [post]
func (h *Handler) Resend(c *webkit.Context) error {
	ctx := c.Context()

	secret := h.config.ResendWebhookSecret
	if secret == "" {
		return api.Error(c, http.StatusBadRequest, "Resend webhook secret is not configured")
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return api.Error(c, http.StatusBadRequest, "Error reading request body")
	}

	headers := email.WebhookHeaders{
		ID:        c.Request.Header.Get("svix-id"),
		Timestamp: c.Request.Header.Get("svix-timestamp"),
		Signature: c.Request.Header.Get("svix-signature"),
	}
	if err = email.VerifyWebhook(string(body), headers, secret); err != nil {
		slog.WarnContext(ctx, "Rejected Resend webhook with invalid signature", "err", err)
		return api.Error(c, http.StatusUnauthorized, "Invalid signature")
	}

	evt, err := email.ParseWebhook(body)
	if err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid payload")
	}

	domainEvt, tracked, err := email.ToEmailEvent(evt, headers.ID)
	if err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid payload")
	}
	if !tracked {
		slog.WarnContext(ctx, "Rejected Resend webhook with invalid event", "evt", evt)
		return api.OK(c, http.StatusOK, nil, "Successfully tracked event")
	}

	if err = h.emailEvents.Process(ctx, domainEvt); err != nil {
		slog.ErrorContext(ctx, "Failed to process Resend webhook event", "type", domainEvt.Type, "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to process event")
	}

	return api.OK(c, http.StatusOK, nil, "Successfully tracked event")
}
