// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"fmt"
	"sort"
	"strconv"
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
// cron at the end of a successful run. The card shows the issue's
// subject + intro + item count and, per platform, the drafted text with
// an "Edit" button deep-linking into the dashboard.
//
// Lives in the digest service (not the slack gateway) because the body
// composition is domain-aware: it knows about digest issues, social
// post kinds, and the dashboard URL. The slack gateway stays a plain
// send-channel.
func buildSummary(in buildSummaryInput) slack.Request {
	header := "📰 Digest " + in.IssueDate + " — drafts ready for review"

	blocks := make([]slack.Block, 0, 4+2*len(in.Drafts))
	blocks = append(blocks, slackgo.NewHeaderBlock(
		slackgo.NewTextBlockObject(slackgo.PlainTextType, header, false, false),
	))

	if body := summaryBody(in); body != "" {
		blocks = append(blocks, slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, body, false, false),
			nil, nil,
		))
	}

	if in.IssueID > 0 {
		blocks = append(blocks, slack.ButtonRow([]slack.LinkButton{{
			Label: "View issue",
			URL:   fmt.Sprintf("%s/issues/%d", env.DashboardURL, in.IssueID),
		}}))
	}

	if len(in.Drafts) > 0 {
		blocks = append(blocks, slackgo.NewDividerBlock())
		for _, group := range groupDrafts(in.Drafts) {
			blocks = append(blocks, slackgo.NewSectionBlock(
				slackgo.NewTextBlockObject(slackgo.MarkdownType,
					fmt.Sprintf("*%s · %s*\n%s", titleCase(group.Kind), titleCase(group.Platform), codeBlock(group.Text)),
					false, false),
				nil, nil,
			))
			blocks = append(blocks, slack.ButtonRow([]slack.LinkButton{{
				Label: "Edit",
				URL:   fmt.Sprintf("%s/social/drafts?id=%d", env.DashboardURL, group.ID),
			}}))
		}
	}

	blocks = append(blocks, slackgo.NewContextBlock("",
		slackgo.NewTextBlockObject(slackgo.MarkdownType,
			"Auto-publishes at 11:00 UTC unless cancelled from the dashboard.",
			false, false),
	))

	fallback := header
	if in.Subject != "" {
		fallback += " — " + in.Subject
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

func summaryBody(in buildSummaryInput) string {
	parts := make([]string, 0, 3)
	if in.Subject != "" {
		parts = append(parts, "*Subject:* "+in.Subject)
	}
	if in.Intro != "" {
		parts = append(parts, "*Intro:* "+in.Intro)
	}
	if in.ItemCount > 0 {
		parts = append(parts, "*Items:* "+strconv.Itoa(in.ItemCount))
	}
	return strings.Join(parts, "\n\n")
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
