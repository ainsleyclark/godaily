// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slackdata

import (
	"strings"

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

func context(text string) slack.Block {
	return slackgo.NewContextBlock("", slackgo.NewTextBlockObject(slackgo.MarkdownType, text, false, false))
}

func section(markdown string) slack.Block {
	return slackgo.NewSectionBlock(slackgo.NewTextBlockObject(slackgo.MarkdownType, markdown, false, false), nil, nil)
}

// sectionWithButton renders a markdown section with a right-aligned
// accessory link button.
func sectionWithButton(markdown, label, url, style string) slack.Block {
	return slackgo.NewSectionBlock(
		slackgo.NewTextBlockObject(slackgo.MarkdownType, markdown, false, false),
		nil, slackgo.NewAccessory(linkButton(label, url, style)),
	)
}

// linkButton builds a link button element pointing at url.
func linkButton(label, url, style string) *slackgo.ButtonBlockElement {
	btn := slackgo.NewButtonBlockElement("", url,
		slackgo.NewTextBlockObject(slackgo.PlainTextType, label, false, false))
	btn.URL = url
	if style != "" {
		btn.Style = slackgo.Style(style)
	}
	return btn
}

// actions wraps one or more button elements in an actions block, letting a
// single card carry several side-by-side links.
func actions(elements ...slackgo.BlockElement) slack.Block {
	return slackgo.NewActionBlock("", elements...)
}

// blockquote prefixes every line so multi-line text renders as one
// continuous Slack blockquote.
func blockquote(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, l := range lines {
		lines[i] = "> " + l
	}
	return strings.Join(lines, "\n")
}

// preview collapses whitespace and truncates text so it fits on a couple of
// lines next to a section accessory without throwing the layout off.
func preview(text string) string {
	const max = 140
	flat := strings.Join(strings.Fields(text), " ")
	if len([]rune(flat)) <= max {
		return flat
	}
	runes := []rune(flat)
	return strings.TrimRight(string(runes[:max]), " ,.;:-") + "…"
}
