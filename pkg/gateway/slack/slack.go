// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slack

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
)

//go:generate mockgen -package=mockslack -destination=../../mocks/slack/Sender.go . Sender

type (
	// Sender defines the method to send messages via the Slack API.
	Sender interface {
		// Send takes a message and sends it to the configured channel.
		// A Client app with the chat:write.public and chat:write permissions must
		// be installed to the workspace.
		//
		// See: https://api.slack.com/
		Send(ctx context.Context, message string) error

		// MustSend is identical to Send but logs an error instead of
		// returning if one occurs.
		MustSend(ctx context.Context, message string)
	}
	// Client implements the Sender interface to send Slack messages.
	Client struct {
		slackSendFunc slackSendFn
		channel       string
	}
	// Field is an alias of a Slack attachment field to attach to the message.
	Field = slack.AttachmentField
	// slackSendFn is the function used for sending to a Client channel,
	// stubbed for testing.
	slackSendFn func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error)
)

// New creates a new Slack client using the given API token and channel.
// For more information about the Slack API token:
//
//	-> https://pkg.go.dev/github.com/slack-go/slack#New
func New(token, channel string) *Client {
	return &Client{
		slackSendFunc: slack.New(token).PostMessageContext,
		channel:       channel,
	}
}

// Send sends a message to the configured Slack channel.
func (c *Client) Send(ctx context.Context, message string) error {
	attachment := slack.Attachment{
		Pretext: "Go Daily - Digest Message",
		Color:   "#0554c6",
		Fields: []Field{
			{Value: message},
		},
	}
	id, timestamp, err := c.slackSendFunc(ctx, c.channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return errors.Wrapf(
			err,
			"failed to send message to Slack channel '%s' at time '%s'",
			id,
			timestamp,
		)
	}
	return nil
}

// MustSend is identical to Send but logs an error instead of returning if one occurs.
func (c *Client) MustSend(ctx context.Context, message string) {
	if err := c.Send(ctx, message); err != nil {
		slog.ErrorContext(ctx, "Slack error: "+err.Error())
	}
}
