// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package slackcard assembles the social notification cards sent to Slack.
//
// It is intentionally free of domain logic: callers in the social service
// map their domain types (post kinds, platforms) into plain strings and
// Rows, and this package turns them into a Block Kit message. The name
// avoids clashing with the pkg/gateway/slack import.
package slackcard

import (
	"strings"

	slackgo "github.com/slack-go/slack"

	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// maxText caps the post copy rendered inside a card so long posts stay
// readable and never approach Slack's 3000-char section limit.
const maxText = 280

// Row is one line in a card: a bold heading, the post copy rendered as a
// blockquote, and an optional accessory button. Any of the three may be
// empty (e.g. a quote-only row in the collapsed variant).
type Row struct {
	Heading string
	Text    string
	Button  *slack.LinkButton
}

// Build assembles the standard social notification: a header, an optional
// context line, a divider, the supplied rows (each a section with an
// accessory button) and any trailing blocks, plus a coloured sidebar
// attachment. fallback drives the plain-text notification preview.
func Build(title, contextLine, fallback, color string, rows []Row, trailing ...slack.Block) slack.Request {
	blocks := make([]slack.Block, 0, 3+len(rows)+len(trailing))
	blocks = append(blocks, slackgo.NewHeaderBlock(
		slackgo.NewTextBlockObject(slackgo.PlainTextType, title, false, false),
	))
	if contextLine != "" {
		blocks = append(blocks, Context(contextLine))
	}
	if len(rows) > 0 || len(trailing) > 0 {
		blocks = append(blocks, slackgo.NewDividerBlock())
	}
	for _, r := range rows {
		blocks = append(blocks, section(r))
	}
	blocks = append(blocks, trailing...)

	return slack.Request{
		Text:        fallback,
		Blocks:      slack.BlockSet{BlockSet: blocks},
		Attachments: []slack.Attachment{{Color: color, Fallback: fallback}},
	}
}

// Context renders a single markdown context line as a block, exported so
// callers can pass it as a trailing block to Build.
func Context(text string) slack.Block {
	return slackgo.NewContextBlock("",
		slackgo.NewTextBlockObject(slackgo.MarkdownType, text, false, false),
	)
}

// section renders one Row as a section block, attaching the button as a
// right-aligned accessory when present.
func section(r Row) slack.Block {
	md := ""
	if r.Heading != "" {
		md = "*" + r.Heading + "*"
	}
	if r.Text != "" {
		if md != "" {
			md += "\n"
		}
		md += blockquote(r.Text)
	}
	textObj := slackgo.NewTextBlockObject(slackgo.MarkdownType, md, false, false)
	if r.Button == nil {
		return slackgo.NewSectionBlock(textObj, nil, nil)
	}
	return slackgo.NewSectionBlock(textObj, nil, slackgo.NewAccessory(linkButton(*r.Button)))
}

// linkButton converts a slack.LinkButton into a button block element for
// use as a section accessory (slack.ButtonRow only emits action blocks).
func linkButton(b slack.LinkButton) *slackgo.ButtonBlockElement {
	btn := slackgo.NewButtonBlockElement("", b.URL,
		slackgo.NewTextBlockObject(slackgo.PlainTextType, b.Label, false, false),
	)
	btn.URL = b.URL
	if b.Style != "" {
		btn.Style = slackgo.Style(b.Style)
	}
	return btn
}

// blockquote renders text as a Slack markdown blockquote, truncated to
// maxText. Every line is prefixed so multi-line posts render as one
// continuous quote rather than only the first line.
func blockquote(text string) string {
	text = truncate(strings.TrimSpace(text), maxText)
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = "> " + l
	}
	return strings.Join(lines, "\n")
}

// truncate shortens s to at most maxRunes runes, appending an ellipsis
// when it trims anything.
func truncate(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return strings.TrimSpace(string(r[:maxRunes])) + "…"
}
