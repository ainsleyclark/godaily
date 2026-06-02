// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package slackdata builds the Slack Block Kit messages for the digest
// service. It owns message layout only; the digest service decides when to
// send and supplies the data. Its name avoids clashing with the
// pkg/gateway/slack send-channel.
package slackdata

import (
	"fmt"
	"sort"
	"strings"

	slackgo "github.com/slack-go/slack"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// Draft describes one drafted social post for the build summary card.
type Draft struct {
	ID       int64
	Kind     string
	Platform string
	Text     string
}

// Summary is the payload BuildSummary renders. IssueSlug powers the "View
// live copy" link to the public site and may be empty when only rotation
// posts were drafted.
type Summary struct {
	IssueDate string
	IssueID   int64
	IssueSlug string
	Subject   string
	Intro     string
	ItemCount int
	Drafts    []Draft
}

// BuildSummary renders the rich card emitted by the digest build cron at
// the end of a successful run. It leads with the issue subject and a "View
// live copy" button linking to the published page on the public site, shows
// the AI intro as a blockquote, and lists each drafted post with an "Edit"
// deep-link into the dashboard.
func BuildSummary(in Summary) slack.Request {
	blocks := make([]slack.Block, 0, 5+len(in.Drafts))
	blocks = append(blocks, header("Digest ready for review"))

	if ctxLine := summaryContext(in); ctxLine != "" {
		blocks = append(blocks, context(ctxLine))
	}

	// Subject line, with a "View live copy" button linking to the live page on
	// the public site once the issue has been built.
	if in.Subject != "" || in.IssueSlug != "" {
		subject := in.Subject
		if subject == "" {
			subject = "Digest drafted"
		}
		if in.IssueSlug != "" {
			blocks = append(blocks, sectionWithButton("*"+subject+"*",
				"View live copy", fmt.Sprintf("%s/issues/%s/", env.AppURL, in.IssueSlug), "primary"))
		} else {
			blocks = append(blocks, section("*"+subject+"*"))
		}
	}

	if in.Intro != "" {
		blocks = append(blocks, section(blockquote(in.Intro)))
	}

	if len(in.Drafts) > 0 {
		blocks = append(blocks, slackgo.NewDividerBlock())
		for _, group := range groupDrafts(in.Drafts) {
			text := fmt.Sprintf("*%s · %s*\n%s",
				titleCase(group.Kind), titleCase(group.Platform), codeBlock(group.Text))
			blocks = append(blocks, sectionWithButton(text,
				"Edit", fmt.Sprintf("%s/social/drafts?id=%d", env.DashboardURL, group.ID), ""))
		}
	}

	blocks = append(blocks, context(
		"Auto-publishes at 11:00 UTC unless cancelled from the dashboard.",
	))

	fallback := "Digest ready for review"
	if in.Subject != "" {
		fallback += " - " + in.Subject
	}
	return message(fallback, slack.ColorInfo, blocks)
}

// summaryContext builds the one-line meta context under the header.
func summaryContext(in Summary) string {
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

func distinctPlatforms(drafts []Draft) int {
	seen := make(map[string]struct{}, len(drafts))
	for _, d := range drafts {
		seen[d.Platform] = struct{}{}
	}
	return len(seen)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// groupDrafts returns drafts in a stable order (kind ASC, platform ASC) so
// the same build emits the same message on re-run.
func groupDrafts(drafts []Draft) []Draft {
	out := append([]Draft(nil), drafts...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Platform < out[j].Platform
	})
	return out
}
