// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"fmt"
	"sort"
	"strings"

	slackgo "github.com/slack-go/slack"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// buildSummaryDraft describes one drafted social post for the build
// summary Slack card. One buildSummary call accepts a slice of these
// covering every kind+platform combination produced by DraftAll.
type buildSummaryDraft struct {
	ID       int64
	Kind     string
	Platform string
	Text     string
}

// buildSummaryInput is the payload buildSummary renders. IssueID powers
// the "View issue" deep-link and may be 0 when only rotation posts were
// drafted.
type buildSummaryInput struct {
	IssueDate string
	IssueID   int64
	Subject   string
	Intro     string
	ItemCount int
	Drafts    []buildSummaryDraft
}

// buildSummary renders the rich Slack card emitted by the digest build
// cron at the end of a successful run. It leads with the issue subject and
// a "View issue" button, shows the AI intro as a blockquote, and lists each
// drafted post with an "Edit" deep-link into the dashboard.
//
// Lives in the digest service (not the slack gateway) because the body
// composition is domain-aware: it knows about digest issues, social
// post kinds, and the dashboard URL. The slack gateway stays a plain
// send-channel.
func buildSummary(in buildSummaryInput) slack.Request {
	blocks := make([]slack.Block, 0, 5+len(in.Drafts))
	blocks = append(blocks, slackgo.NewHeaderBlock(
		slackgo.NewTextBlockObject(slackgo.PlainTextType, "Digest ready for review", false, false),
	))

	if ctxLine := summaryContext(in); ctxLine != "" {
		blocks = append(blocks, slackgo.NewContextBlock("",
			slackgo.NewTextBlockObject(slackgo.MarkdownType, ctxLine, false, false),
		))
	}

	// Subject line, with a "View issue" button when the issue exists.
	if in.Subject != "" || in.IssueID > 0 {
		subject := in.Subject
		if subject == "" {
			subject = "Digest drafted"
		}
		var acc *slack.Accessory
		if in.IssueID > 0 {
			acc = accessoryButton("View issue", fmt.Sprintf("%s/issues/%d", env.DashboardURL, in.IssueID), "primary")
		}
		blocks = append(blocks, slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, "*"+subject+"*", false, false),
			nil, acc,
		))
	}

	if in.Intro != "" {
		blocks = append(blocks, slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, blockquote(in.Intro), false, false),
			nil, nil,
		))
	}

	if len(in.Drafts) > 0 {
		blocks = append(blocks, slackgo.NewDividerBlock())
		for _, group := range groupDrafts(in.Drafts) {
			blocks = append(blocks, slackgo.NewSectionBlock(
				slackgo.NewTextBlockObject(slackgo.MarkdownType,
					fmt.Sprintf("*%s · %s*\n%s", titleCase(group.Kind), titleCase(group.Platform), codeBlock(group.Text)),
					false, false),
				nil,
				accessoryButton("Edit", fmt.Sprintf("%s/social/drafts?id=%d", env.DashboardURL, group.ID), ""),
			))
		}
	}

	blocks = append(blocks, slackgo.NewContextBlock("",
		slackgo.NewTextBlockObject(slackgo.MarkdownType,
			"Auto-publishes at 11:00 UTC unless cancelled from the dashboard.",
			false, false),
	))

	fallback := "Digest ready for review"
	if in.Subject != "" {
		fallback += " - " + in.Subject
	}

	return slack.Request{
		Text:   fallback,
		Blocks: slack.BlockSet{BlockSet: blocks},
		Attachments: []slack.Attachment{{
			Color:    slack.ColorInfo,
			Fallback: fallback,
		}},
	}
}

// summaryContext builds the one-line meta context under the header:
// date, story count and draft/platform spread.
func summaryContext(in buildSummaryInput) string {
	parts := make([]string, 0, 3)
	if in.IssueDate != "" {
		parts = append(parts, "*"+in.IssueDate+"*")
	}
	if in.ItemCount > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", in.ItemCount, plural(in.ItemCount, "story", "stories")))
	}
	if n := len(in.Drafts); n > 0 {
		p := distinctPlatforms(in.Drafts)
		parts = append(parts, fmt.Sprintf("%d %s across %d %s",
			n, plural(n, "draft", "drafts"), p, plural(p, "platform", "platforms")))
	}
	return strings.Join(parts, "  ·  ")
}

// accessoryButton builds a section accessory link button.
func accessoryButton(label, url, style string) *slack.Accessory {
	btn := slackgo.NewButtonBlockElement("", url,
		slackgo.NewTextBlockObject(slackgo.PlainTextType, label, false, false),
	)
	btn.URL = url
	if style != "" {
		btn.Style = slackgo.Style(style)
	}
	return slackgo.NewAccessory(btn)
}

// distinctPlatforms counts the unique platforms across a draft slice.
func distinctPlatforms(drafts []buildSummaryDraft) int {
	seen := make(map[string]struct{}, len(drafts))
	for _, d := range drafts {
		seen[d.Platform] = struct{}{}
	}
	return len(seen)
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

// plural returns one when n == 1 and many otherwise.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// codeBlock wraps text in a Slack fenced code block so multi-line drafts
// render with monospace + their own border.
func codeBlock(text string) string {
	if text == "" {
		return ""
	}
	return "```\n" + text + "\n```"
}

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// groupDrafts returns drafts in a stable order (kind ASC, platform ASC)
// so the same build emits the same Slack message on re-run.
func groupDrafts(drafts []buildSummaryDraft) []buildSummaryDraft {
	out := append([]buildSummaryDraft(nil), drafts...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Platform < out[j].Platform
	})
	return out
}
