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

// Resend handles POST /webhooks/resend.
// The endpoint is public but every request is verified against the Svix-style
// signature Resend includes; unsigned or tampered requests are rejected.
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
		return webkit.NewError(http.StatusBadRequest, "invalid payload")
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
