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

// Attachment sidebar colours used by the builders in builders.go. Callers
// composing a Request by hand can reuse the same palette for consistency.
const (
	ColorSuccess = "#36a64f"
	ColorError   = "#e01e5a"
	ColorInfo    = "#0554c6"
	ColorWarn    = "#ecb22e"
)

type (
	// Sender defines the method to send messages via the Slack API.
	Sender interface {
		// Send posts a message to the configured channel. A client app
		// with the chat:write.public and chat:write permissions must be
		// installed to the workspace.
		//
		// See: https://api.slack.com/methods/chat.postMessage
		Send(ctx context.Context, req Request) error

		// MustSend is identical to Send but logs an error instead of
		// returning if one occurs.
		MustSend(ctx context.Context, req Request)
	}

	// Client implements the Sender interface to send Slack messages.
	Client struct {
		slackSendFunc slackSendFn
		channel       string
	}

	// Request mirrors the body of Slack's chat.postMessage API. Compose
	// it with the helpers in builders.go (Plain, Info, Success, Warn,
	// Error) or construct one by hand using the type aliases below.
	//
	// The Channel field is ignored — Client.Send always posts to the
	// channel configured at construction time.
	Request = slack.Msg

	// Re-exports of slack-go primitives so callers can build rich
	// payloads without importing github.com/slack-go/slack directly.
	Block      = slack.Block
	BlockSet   = slack.Blocks
	Attachment = slack.Attachment
	Field      = slack.AttachmentField
	Section    = slack.SectionBlock
	Header     = slack.HeaderBlock
	Action     = slack.ActionBlock
	ContextBk  = slack.ContextBlock
	Divider    = slack.DividerBlock
	Button     = slack.ButtonBlockElement
	TextObject = slack.TextBlockObject
	Accessory  = slack.Accessory

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

// Send posts a Request to the configured Slack channel.
func (c *Client) Send(ctx context.Context, req Request) error {
	opts := requestToOptions(req)
	id, timestamp, err := c.slackSendFunc(ctx, c.channel, opts...)
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
func (c *Client) MustSend(ctx context.Context, req Request) {
	if err := c.Send(ctx, req); err != nil {
		slog.ErrorContext(ctx, "Slack error: "+err.Error())
	}
}

// requestToOptions converts a Request into the MsgOption slice that
// slack-go's PostMessageContext expects. Empty fields are skipped so the
// outbound payload matches what was set on the Request.
func requestToOptions(req Request) []slack.MsgOption {
	opts := make([]slack.MsgOption, 0, 6)
	if req.Text != "" {
		opts = append(opts, slack.MsgOptionText(req.Text, false))
	}
	if len(req.Blocks.BlockSet) > 0 {
		opts = append(opts, slack.MsgOptionBlocks(req.Blocks.BlockSet...))
	}
	if len(req.Attachments) > 0 {
		opts = append(opts, slack.MsgOptionAttachments(req.Attachments...))
	}
	if req.ThreadTimestamp != "" {
		opts = append(opts, slack.MsgOptionTS(req.ThreadTimestamp))
	}
	// Operational messages embed URLs purely as deep-links; the auto-unfurled
	// previews add visual noise and push the actual content off-screen.
	opts = append(opts,
		slack.MsgOptionDisableLinkUnfurl(),
		slack.MsgOptionDisableMediaUnfurl(),
	)
	return opts
}
