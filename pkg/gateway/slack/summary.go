// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slack

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/slack-go/slack"
)

// DashboardURL is the public base URL of the GoDaily admin dashboard.
// Hard-coded rather than threaded through config: there is exactly one
// dashboard per deployment and the URL never differs by environment.
const DashboardURL = "https://godaily.dev/dashboard"

// BuildSummaryDraft describes one drafted social post for the build
// summary Slack card. One BuildSummary call accepts a slice of these
// covering every kind+platform combination produced by DraftAll.
type BuildSummaryDraft struct {
	ID       int64
	Kind     string
	Platform string
	Text     string
}

// BuildSummaryInput is the payload BuildSummary renders. IssueID powers
// the "View issue" deep-link and may be 0 when only rotation posts were
// drafted.
type BuildSummaryInput struct {
	IssueDate string
	IssueID   int64
	Subject   string
	Intro     string
	ItemCount int
	Drafts    []BuildSummaryDraft
}

// BuildSummary renders the rich Slack card emitted by the digest build
// cron at the end of a successful run. The card shows the issue's
// subject + intro + item count and, per platform, the drafted text with
// an "Edit" button deep-linking into the dashboard.
//
// The publish cron at 11:00 promotes every draft, so the operator's
// review window is the gap between build (02:00) and publish (11:00).
func BuildSummary(in BuildSummaryInput) Request {
	header := "📰 Digest " + in.IssueDate + " — drafts ready for review"

	blocks := make([]Block, 0, 4+2*len(in.Drafts))
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, header, false, false),
	))

	summary := summaryBody(in)
	if summary != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, summary, false, false),
			nil, nil,
		))
	}

	if in.IssueID > 0 {
		blocks = append(blocks, buttonRow([]LinkButton{{
			Label: "View issue",
			URL:   fmt.Sprintf("%s/issues/%d", DashboardURL, in.IssueID),
		}}))
	}

	if len(in.Drafts) > 0 {
		blocks = append(blocks, slack.NewDividerBlock())
		for _, group := range groupDrafts(in.Drafts) {
			blocks = append(blocks, slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType,
					fmt.Sprintf("*%s · %s*\n%s", titleCase(group.Kind), titleCase(group.Platform), codeBlock(group.Text)),
					false, false),
				nil, nil,
			))
			blocks = append(blocks, buttonRow([]LinkButton{{
				Label: "Edit",
				URL:   fmt.Sprintf("%s/social/drafts?id=%d", DashboardURL, group.ID),
			}}))
		}
	}

	blocks = append(blocks, slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType,
			"Auto-publishes at 11:00 UTC unless cancelled from the dashboard.",
			false, false),
	))

	fallback := header
	if in.Subject != "" {
		fallback += " — " + in.Subject
	}

	return Request{
		Text:   fallback,
		Blocks: BlockSet{BlockSet: blocks},
		Attachments: []Attachment{{
			Color:    ColorInfo,
			Fallback: fallback,
		}},
	}
}

func summaryBody(in BuildSummaryInput) string {
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
func groupDrafts(drafts []BuildSummaryDraft) []BuildSummaryDraft {
	out := append([]BuildSummaryDraft(nil), drafts...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Platform < out[j].Platform
	})
	return out
}
