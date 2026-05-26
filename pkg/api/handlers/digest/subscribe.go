// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleydev/webkit/pkg/webkit"
)

// Subscribe handles POST /subscribe.
func (h *Handler) Subscribe(c *webkit.Context) error {
	ctx := c.Context()

	var body struct {
		Email string `json:"email"`
	}
	if err := c.BindJSON(&body); err != nil || body.Email == "" {
		return webkit.NewError(http.StatusBadRequest, "email is required")
	}

	if _, err := mail.ParseAddress(body.Email); err != nil {
		return webkit.NewError(http.StatusBadRequest, "invalid email address")
	}

	if _, err := h.subscribers.Subscribe(ctx, body.Email); err != nil {
		if errors.Is(err, audience.ErrAlreadySubscribed) {
			return webkit.NewError(http.StatusConflict, "already subscribed")
		}
		return webkit.NewError(http.StatusInternalServerError, err.Error())
	}

	msg := "New subscriber: " + body.Email
	if count, err := h.subscribersRepo.CountActive(ctx); err == nil {
		msg += fmt.Sprintf(" | Total subscribers: %d", count)
	}
	h.slack.MustSend(ctx, msg)

	return c.NoContent(http.StatusOK)
}
