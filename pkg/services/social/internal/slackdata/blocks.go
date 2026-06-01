// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slackdata

import (
	"strconv"
	"strings"

	slackgo "github.com/slack-go/slack"

	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// maxText caps the post copy rendered inside a section blockquote so long
// posts stay readable and never approach Slack's 3000-char section limit.
const maxText = 280

// linkButton is a label + URL + optional style ("primary"/"danger"/"").
type linkButton struct {
	label string
	url   string
	style string
}

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
func sectionWithButton(markdown string, btn linkButton) slack.Block {
	return slackgo.NewSectionBlock(
		slackgo.NewTextBlockObject(slackgo.MarkdownType, markdown, false, false),
		nil, slackgo.NewAccessory(button(btn)),
	)
}

// buttonRow renders a single action block with one button per entry.
func buttonRow(btns []linkButton) slack.Block {
	elements := make([]slackgo.BlockElement, 0, len(btns))
	for i, b := range btns {
		el := button(b)
		el.ActionID = "link_" + strconv.Itoa(i)
		elements = append(elements, el)
	}
	return slackgo.NewActionBlock("", elements...)
}

func button(b linkButton) *slackgo.ButtonBlockElement {
	btn := slackgo.NewButtonBlockElement("", b.url,
		slackgo.NewTextBlockObject(slackgo.PlainTextType, b.label, false, false))
	btn.URL = b.url
	if b.style != "" {
		btn.Style = slackgo.Style(b.style)
	}
	return btn
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

// truncate shortens s to at most maxRunes runes, appending an ellipsis when
// it trims anything.
func truncate(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return strings.TrimSpace(string(r[:maxRunes])) + "…"
}
