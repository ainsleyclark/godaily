// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"strings"

	slackgo "github.com/slack-go/slack"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// maxCardText caps the post copy rendered inside a Slack card so long
// posts stay readable and never approach Slack's 3000-char section limit.
const maxCardText = 280

// cardRow is one line in a social Slack card: a bold heading, the post
// copy rendered as a blockquote, and an optional accessory button. Any of
// the three may be empty (e.g. a quote-only row in the collapsed variant).
type cardRow struct {
	heading string
	text    string
	button  *slack.LinkButton
}

// socialCard assembles the standard social notification: a header, an
// optional context line, a divider, the supplied rows (each a section with
// an accessory button) and any trailing blocks, plus a coloured sidebar
// attachment. fallback drives the plain-text notification preview.
func socialCard(title, contextLine, fallback, color string, rows []cardRow, trailing ...slack.Block) slack.Request {
	blocks := make([]slack.Block, 0, 3+len(rows)+len(trailing))
	blocks = append(blocks, slackgo.NewHeaderBlock(
		slackgo.NewTextBlockObject(slackgo.PlainTextType, title, false, false),
	))
	if contextLine != "" {
		blocks = append(blocks, contextBlock(contextLine))
	}
	if len(rows) > 0 || len(trailing) > 0 {
		blocks = append(blocks, slackgo.NewDividerBlock())
	}
	for _, r := range rows {
		blocks = append(blocks, sectionRow(r))
	}
	blocks = append(blocks, trailing...)

	return slack.Request{
		Text:        fallback,
		Blocks:      slack.BlockSet{BlockSet: blocks},
		Attachments: []slack.Attachment{{Color: color, Fallback: fallback}},
	}
}

// sectionRow renders one cardRow as a section block, attaching the button
// as a right-aligned accessory when present.
func sectionRow(r cardRow) slack.Block {
	md := ""
	if r.heading != "" {
		md = "*" + r.heading + "*"
	}
	if r.text != "" {
		if md != "" {
			md += "\n"
		}
		md += blockquote(r.text)
	}
	textObj := slackgo.NewTextBlockObject(slackgo.MarkdownType, md, false, false)
	if r.button == nil {
		return slackgo.NewSectionBlock(textObj, nil, nil)
	}
	return slackgo.NewSectionBlock(textObj, nil, slackgo.NewAccessory(linkButton(*r.button)))
}

// contextBlock renders a single markdown context line.
func contextBlock(text string) slack.Block {
	return slackgo.NewContextBlock("",
		slackgo.NewTextBlockObject(slackgo.MarkdownType, text, false, false),
	)
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
// maxCardText. Every line is prefixed so multi-line posts render as one
// continuous quote rather than only the first line.
func blockquote(text string) string {
	text = truncate(strings.TrimSpace(text), maxCardText)
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

// kindLabel renders a PostKind as a human title for card headings, e.g.
// "new_source" -> "New source".
func kindLabel(k social.PostKind) string {
	s := strings.ReplaceAll(string(k), "_", " ")
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// plural returns one when n == 1 and many otherwise.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
