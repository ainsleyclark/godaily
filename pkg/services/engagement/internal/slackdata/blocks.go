// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slackdata

import (
	slackgo "github.com/slack-go/slack"

	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// message wraps a block slice and a single coloured attachment into a
// Request. fallback drives the plain-text notification preview.
func message(fallback, color string, blocks []slack.Block) slack.Request {
	return slack.Request{
		Text:        fallback,
		Blocks:      slack.BlockSet{BlockSet: blocks},
		Attachments: []slack.Attachment{{Color: color, Fallback: fallback}},
	}
}

func header(text string) slack.Block {
	return slackgo.NewHeaderBlock(slackgo.NewTextBlockObject(slackgo.PlainTextType, text, false, false))
}

func divider() slack.Block { return slackgo.NewDividerBlock() }

func context(text string) slack.Block {
	return slackgo.NewContextBlock("", slackgo.NewTextBlockObject(slackgo.MarkdownType, text, false, false))
}

func section(markdown string) slack.Block {
	return slackgo.NewSectionBlock(slackgo.NewTextBlockObject(slackgo.MarkdownType, markdown, false, false), nil, nil)
}

// sectionWithButton renders a markdown section with a right-aligned
// accessory link button.
func sectionWithButton(markdown, label, url, style string) slack.Block {
	btn := slackgo.NewButtonBlockElement("", url,
		slackgo.NewTextBlockObject(slackgo.PlainTextType, label, false, false))
	btn.URL = url
	if style != "" {
		btn.Style = slackgo.Style(style)
	}
	return slackgo.NewSectionBlock(
		slackgo.NewTextBlockObject(slackgo.MarkdownType, markdown, false, false),
		nil, slackgo.NewAccessory(btn),
	)
}

// fields renders a titled section whose body is a list of two-column
// markdown fields.
func fields(title string, vals []string) slack.Block {
	objs := make([]*slackgo.TextBlockObject, len(vals))
	for i, f := range vals {
		objs[i] = slackgo.NewTextBlockObject(slackgo.MarkdownType, f, false, false)
	}
	return slackgo.NewSectionBlock(
		slackgo.NewTextBlockObject(slackgo.MarkdownType, title, false, false), objs, nil)
}
