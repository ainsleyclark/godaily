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

// Subscribe handles POST /subscribe.
func (h *Handler) Subscribe(c *webkit.Context) error {
	ctx := c.Context()

	var body struct {
		Email string `json:"email"`
	}
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
