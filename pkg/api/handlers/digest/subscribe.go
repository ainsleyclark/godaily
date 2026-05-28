// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"

	"github.com/ainsleydev/webkit/pkg/webkit"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
)

// subscribeRequest is the body for POST /subscribe.
type subscribeRequest struct {
	Email string `json:"email" example:"hello@ainsley.dev"`
} //@name SubscribeRequest

// Subscribe godoc
//
//	@Summary		Subscribe an email address.
//	@Description	Registers an email address for the daily digest and sends a confirmation email.
//	@Tags			subscription
//	@Accept			json
//	@Produce		json
//	@Param			request	body		subscribeRequest	true	"Subscription request"
//	@Success		200		{object}	api.MessageResponse		"Successfully subscribed"
//	@Failure		400		{object}	api.MessageResponse		"Email is required or invalid"
//	@Failure		409		{object}	api.MessageResponse		"Already subscribed"
//	@Failure		500		{object}	api.MessageResponse		"Failed to subscribe"
//	@Router			/subscribe [post]
func (h *Handler) Subscribe(c *webkit.Context) error {
	ctx := c.Context()

	var body subscribeRequest
	if err := c.BindJSON(&body); err != nil || body.Email == "" {
		return api.Error(c, http.StatusBadRequest, "Email is required")
	}

	if _, err := mail.ParseAddress(body.Email); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid email address")
	}

	if _, err := h.subscribers.Subscribe(ctx, body.Email); err != nil {
		if errors.Is(err, audience.ErrAlreadySubscribed) {
			return api.Error(c, http.StatusConflict, "Already subscribed")
		}
		slog.ErrorContext(ctx, "Failed to subscribe", "err", err)
		return api.Error(c, http.StatusInternalServerError, "Failed to subscribe")
	}

	msg := "New subscriber: " + body.Email
	if count, err := h.subscribersRepo.CountActive(ctx); err == nil {
		msg += fmt.Sprintf(" | Total subscribers: %d", count)
	}
	h.slack.MustSend(ctx, msg)

	return api.OK(c, http.StatusOK, nil, "Successfully subscribed")
}
