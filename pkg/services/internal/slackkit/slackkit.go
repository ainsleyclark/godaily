// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package slackkit assembles Slack Block Kit messages for the service
// layer. It is deliberately free of domain logic: callers map their domain
// types into plain strings + Rows and this package turns them into blocks.
//
// It lives under pkg/services/internal so every service package (social,
// digest, engagement) can share it, while the pkg/gateway/slack package
// stays a pure send-channel.
package slackkit

import (
	"strings"

	slackgo "github.com/slack-go/slack"

	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// maxText caps the post copy rendered inside a Row blockquote so long posts
// stay readable and never approach Slack's 3000-char section limit.
const maxText = 280

// Row is one line in the per-platform card built by Build: a bold heading,
// the copy rendered as a (truncated) blockquote, and an optional accessory
// button. Any of the three may be empty.
type Row struct {
	Heading string
	Text    string
	Button  *slack.LinkButton
}

// Message wraps a block slice and a single coloured attachment into a
// Request. fallback drives the plain-text notification preview.
func Message(fallback, color string, blocks []slack.Block) slack.Request {
	return slack.Request{
		Text:        fallback,
		Blocks:      slack.BlockSet{BlockSet: blocks},
		Attachments: []slack.Attachment{{Color: color, Fallback: fallback}},
	}
}

// Build assembles a header + optional context + divider + per-Row sections
// + any trailing blocks into a coloured card. Used for the social
// post-published / drafts-published notifications.
func Build(title, contextLine, fallback, color string, rows []Row, trailing ...slack.Block) slack.Request {
	blocks := make([]slack.Block, 0, 3+len(rows)+len(trailing))
	blocks = append(blocks, Header(title))
	if contextLine != "" {
		blocks = append(blocks, Context(contextLine))
	}
	if len(rows) > 0 || len(trailing) > 0 {
		blocks = append(blocks, Divider())
	}
	for _, r := range rows {
		blocks = append(blocks, row(r))
	}
	blocks = append(blocks, trailing...)
	return Message(fallback, color, blocks)
}

// Header renders a plain-text header block.
func Header(text string) slack.Block {
	return slackgo.NewHeaderBlock(slackgo.NewTextBlockObject(slackgo.PlainTextType, text, false, false))
}

// Divider renders a divider block.
func Divider() slack.Block { return slackgo.NewDividerBlock() }

// Context renders a single markdown context line.
func Context(text string) slack.Block {
	return slackgo.NewContextBlock("", slackgo.NewTextBlockObject(slackgo.MarkdownType, text, false, false))
}

// Section renders a markdown section block.
func Section(markdown string) slack.Block {
	return slackgo.NewSectionBlock(slackgo.NewTextBlockObject(slackgo.MarkdownType, markdown, false, false), nil, nil)
}

// SectionWithButton renders a markdown section with a right-aligned
// accessory link button.
func SectionWithButton(markdown string, btn slack.LinkButton) slack.Block {
	return slackgo.NewSectionBlock(
		slackgo.NewTextBlockObject(slackgo.MarkdownType, markdown, false, false),
		nil, slackgo.NewAccessory(button(btn)),
	)
}

// Fields renders a section whose body is a list of two-column markdown
// fields, with an optional bold title above them.
func Fields(title string, fields []string) slack.Block {
	objs := make([]*slackgo.TextBlockObject, len(fields))
	for i, f := range fields {
		objs[i] = slackgo.NewTextBlockObject(slackgo.MarkdownType, f, false, false)
	}
	var titleObj *slackgo.TextBlockObject
	if title != "" {
		titleObj = slackgo.NewTextBlockObject(slackgo.MarkdownType, title, false, false)
	}
	return slackgo.NewSectionBlock(titleObj, objs, nil)
}

// Blockquote prefixes every line so multi-line text renders as one
// continuous Slack blockquote. It does not truncate; use Truncate first
// when the source is unbounded.
func Blockquote(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, l := range lines {
		lines[i] = "> " + l
	}
	return strings.Join(lines, "\n")
}

// CodeBlock wraps text in a fenced code block so multi-line content renders
// monospaced with its own border.
func CodeBlock(text string) string {
	if text == "" {
		return ""
	}
	return "```\n" + text + "\n```"
}

// Truncate shortens s to at most maxRunes runes, appending an ellipsis when
// it trims anything.
func Truncate(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return strings.TrimSpace(string(r[:maxRunes])) + "…"
}

// row renders a Row: heading + truncated blockquote + optional accessory.
func row(r Row) slack.Block {
	md := ""
	if r.Heading != "" {
		md = "*" + r.Heading + "*"
	}
	if r.Text != "" {
		if md != "" {
			md += "\n"
		}
		md += Blockquote(Truncate(r.Text, maxText))
	}
	if r.Button == nil {
		return Section(md)
	}
	return SectionWithButton(md, *r.Button)
}

// button converts a slack.LinkButton into a button block element.
func button(b slack.LinkButton) *slackgo.ButtonBlockElement {
	btn := slackgo.NewButtonBlockElement("", b.URL,
		slackgo.NewTextBlockObject(slackgo.PlainTextType, b.Label, false, false),
	)
	btn.URL = b.URL
	if b.Style != "" {
		btn.Style = slackgo.Style(b.Style)
	}
	return btn
}
