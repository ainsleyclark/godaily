// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/internal/slackkit"
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
// The body composition is domain-aware (it knows about digest issues,
// social post kinds, and the dashboard URL); the block-kit plumbing comes
// from the shared slackkit package.
func buildSummary(in buildSummaryInput) slack.Request {
	blocks := make([]slack.Block, 0, 5+len(in.Drafts))
	blocks = append(blocks, slackkit.Header("Digest ready for review"))

	if ctxLine := summaryContext(in); ctxLine != "" {
		blocks = append(blocks, slackkit.Context(ctxLine))
	}

	// Subject line, with a "View issue" button when the issue exists.
	if in.Subject != "" || in.IssueID > 0 {
		subject := in.Subject
		if subject == "" {
			subject = "Digest drafted"
		}
		if in.IssueID > 0 {
			blocks = append(blocks, slackkit.SectionWithButton("*"+subject+"*", slack.LinkButton{
				Label: "View issue",
				URL:   fmt.Sprintf("%s/issues/%d", env.DashboardURL, in.IssueID),
				Style: "primary",
			}))
		} else {
			blocks = append(blocks, slackkit.Section("*"+subject+"*"))
		}
	}

	if in.Intro != "" {
		blocks = append(blocks, slackkit.Section(slackkit.Blockquote(in.Intro)))
	}

	if len(in.Drafts) > 0 {
		blocks = append(blocks, slackkit.Divider())
		for _, group := range groupDrafts(in.Drafts) {
			text := fmt.Sprintf("*%s · %s*\n%s",
				titleCase(group.Kind), titleCase(group.Platform), slackkit.CodeBlock(group.Text))
			blocks = append(blocks, slackkit.SectionWithButton(text, slack.LinkButton{
				Label: "Edit",
				URL:   fmt.Sprintf("%s/social/drafts?id=%d", env.DashboardURL, group.ID),
			}))
		}
	}

	blocks = append(blocks, slackkit.Context(
		"Auto-publishes at 11:00 UTC unless cancelled from the dashboard."))

	fallback := "Digest ready for review"
	if in.Subject != "" {
		fallback += " - " + in.Subject
	}
	return slackkit.Message(fallback, slack.ColorInfo, blocks)
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

// distinctPlatforms counts the unique platforms across a draft slice.
func distinctPlatforms(drafts []buildSummaryDraft) int {
	seen := make(map[string]struct{}, len(drafts))
	for _, d := range drafts {
		seen[d.Platform] = struct{}{}
	}
	return len(seen)
}

// plural returns one when n == 1 and many otherwise.
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
