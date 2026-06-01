// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slack

import (
	"strconv"

	"github.com/slack-go/slack"
)

// LinkButton is a shorthand for a URL action-button element. Pass a slice
// of these to Info / Success to render a row of clickable buttons.
type LinkButton struct {
	Label string
	URL   string
	// Style is "primary", "danger" or "" (default).
	Style string
}

// Plain produces a Request whose only field is Text. It is the migration
// shim for callers that don't need block-kit formatting yet.
func Plain(text string) Request {
	return Request{Text: text}
}

// Info builds a neutral (blue) message with an optional row of buttons.
func Info(title, body string, btns ...LinkButton) Request {
	return build(title, body, ColorInfo, btns)
}

// Success builds a green message — use it for "X happened" notifications
// like a new subscriber or a social post going live.
func Success(title, body string, btns ...LinkButton) Request {
	return build(title, body, ColorSuccess, btns)
}

// Warn builds a yellow message for non-fatal warnings (e.g. partial
// failures during a collection run).
func Warn(title, body string) Request {
	return build(title, body, ColorWarn, nil)
}

// Error builds a red message from an error. The err is rendered as the
// body of the section block.
func Error(title string, err error) Request {
	body := ""
	if err != nil {
		body = err.Error()
	}
	return build(title, body, ColorError, nil)
}

// build assembles the Request used by every coloured builder. The header
// + section + optional action blocks render the rich card; the coloured
// attachment provides the sidebar bar; Text is the notification fallback.
func build(title, body, color string, btns []LinkButton) Request {
	blocks := make([]Block, 0, 3)
	if title != "" {
		blocks = append(blocks, slack.NewHeaderBlock(
			slack.NewTextBlockObject(slack.PlainTextType, title, false, false),
		))
	}
	if body != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, body, false, false),
			nil, nil,
		))
	}
	if len(btns) > 0 {
		blocks = append(blocks, ButtonRow(btns))
	}

	fallback := title
	if body != "" {
		if fallback != "" {
			fallback += " — "
		}
		fallback += body
	}

	return Request{
		Text:   fallback,
		Blocks: BlockSet{BlockSet: blocks},
		Attachments: []Attachment{{
			Color:    color,
			Fallback: fallback,
		}},
	}
}

// ButtonRow turns a slice of LinkButtons into a single Slack action
// block. Exported so service-layer builders that compose richer messages
// (interleaved section + action rows) can build buttons without
// re-importing slack-go directly.
func ButtonRow(btns []LinkButton) *Action {
	elements := make([]slack.BlockElement, 0, len(btns))
	for i, b := range btns {
		btn := slack.NewButtonBlockElement(
			"link_"+strconv.Itoa(i),
			b.URL,
			slack.NewTextBlockObject(slack.PlainTextType, b.Label, false, false),
		)
		btn.URL = b.URL
		if b.Style != "" {
			btn.Style = slack.Style(b.Style)
		}
		elements = append(elements, btn)
	}
	return slack.NewActionBlock("", elements...)
}
