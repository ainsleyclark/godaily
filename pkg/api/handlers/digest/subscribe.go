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
	slackgo "github.com/slack-go/slack"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
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

	fields := []*slackgo.TextBlockObject{
		slackgo.NewTextBlockObject(slackgo.MarkdownType, "*Email:*\n"+body.Email, false, false),
	}
	if count, err := h.subscribersRepo.CountActive(ctx); err == nil {
		fields = append(fields, slackgo.NewTextBlockObject(slackgo.MarkdownType,
			fmt.Sprintf("*Total active:*\n%d", count), false, false))
	}
	fallback := "New subscriber: " + body.Email
	h.slack.MustSend(ctx, slack.Request{
		Text: fallback,
		Blocks: slack.BlockSet{BlockSet: []slack.Block{
			slackgo.NewHeaderBlock(slackgo.NewTextBlockObject(slackgo.PlainTextType, "New subscriber", false, false)),
			slackgo.NewSectionBlock(nil, fields, nil),
			slackgo.NewContextBlock("", slackgo.NewTextBlockObject(slackgo.MarkdownType,
				"Pending confirmation  ·  double opt-in email sent", false, false)),
		}},
		Attachments: []slack.Attachment{{Color: slack.ColorSuccess, Fallback: fallback}},
	})

	return api.OK(c, http.StatusOK, nil, "Successfully subscribed")
}
