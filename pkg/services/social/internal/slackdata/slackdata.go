// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package slackdata builds the Slack Block Kit messages for the social
// service. It owns message layout and platform/kind labelling only; the
// social service decides when to send and supplies the post results. Its
// name avoids clashing with the pkg/gateway/slack send-channel.
package slackdata

import (
	"fmt"
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// PostPublished builds the "social post published" card for one publish
// run: one section per platform that went live showing the copy and a
// "View on" button, collapsing to a single quote + button row when every
// platform shares identical copy. It returns ok=false when no platform
// actually went live.
func PostPublished(kind social.PostKind, subject string, issueID *int64, results []social.PostResult) (slack.Request, bool) {
	posted := live(results)
	if len(posted) == 0 {
		return slack.Request{}, false
	}

	title := "Social post published: " + kindLabel(kind)
	fallback := title
	if subject != "" {
		fallback += " - " + subject
	}
	closing := context(fmt.Sprintf(
		"Posted to %d %s  ·  <https://godaily.dev|godaily.dev>",
		len(posted), plural(len(posted), "platform", "platforms")))

	blocks := []slack.Block{header(title)}
	if ctxLine := issueContext(subject, issueID); ctxLine != "" {
		blocks = append(blocks, context(ctxLine))
	}
	blocks = append(blocks, divider())

	// Auto-collapse when every platform shares the same copy: one quote
	// plus a button row instead of repeating identical text per platform.
	if sameCopy(posted) {
		blocks = append(blocks, section(blockquote(truncate(posted[0].Text, maxText))))
		btns := make([]linkButton, 0, len(posted))
		for _, r := range posted {
			btns = append(btns, linkButton{"View on " + PlatformLabel(r.Platform), r.PostURL, "primary"})
		}
		blocks = append(blocks, buttonRow(btns), closing)
		return message(fallback, slack.ColorSuccess, blocks), true
	}

	for _, r := range posted {
		blocks = append(blocks, sectionWithButton(
			"*"+PlatformLabel(r.Platform)+"*\n"+blockquote(truncate(r.Text, maxText)),
			linkButton{"View on " + PlatformLabel(r.Platform), r.PostURL, "primary"}))
	}
	blocks = append(blocks, closing)
	return message(fallback, slack.ColorSuccess, blocks), true
}

// DraftsPublished builds the "social drafts published" card: one section
// per published draft (kind, platform, copy) with a "View post" button. It
// returns ok=false when nothing went live.
func DraftsPublished(date time.Time, results []social.PostResult) (slack.Request, bool) {
	posted := live(results)
	if len(posted) == 0 {
		return slack.Request{}, false
	}

	blocks := make([]slack.Block, 0, 3+len(posted))
	day := date.Format("2006-01-02")
	title := "Social drafts published"
	blocks = append(blocks,
		header(title),
		context(fmt.Sprintf("%d %s now live for *%s*", len(posted), plural(len(posted), "post", "posts"), day)),
		divider(),
	)
	for _, r := range posted {
		heading := PlatformLabel(r.Platform)
		if k := kindLabel(r.Kind); k != "" {
			heading = k + "  ·  " + heading
		}
		blocks = append(blocks, sectionWithButton(
			"*"+heading+"*\n"+blockquote(truncate(r.Text, maxText)),
			linkButton{"View post", r.PostURL, "primary"}))
	}

	fallback := fmt.Sprintf("%s - %d post(s) live for %s", title, len(posted), day)
	return message(fallback, slack.ColorSuccess, blocks), true
}

// live returns the results that actually went live (no error, not skipped,
// has a URL).
func live(results []social.PostResult) []social.PostResult {
	out := make([]social.PostResult, 0, len(results))
	for _, r := range results {
		if r.Err != nil || r.Skipped || r.PostURL == "" {
			continue
		}
		out = append(out, r)
	}
	return out
}

// issueContext builds the issue context line: the subject and, when known,
// a link to the issue in the dashboard.
func issueContext(subject string, issueID *int64) string {
	parts := make([]string, 0, 2)
	if subject != "" {
		parts = append(parts, "*Issue:* "+subject)
	}
	if issueID != nil {
		parts = append(parts, fmt.Sprintf("<%s/issues/%d|Issue #%d>", env.DashboardURL, *issueID, *issueID))
	}
	return strings.Join(parts, "  ·  ")
}

// sameCopy reports whether every result carries identical post text.
func sameCopy(results []social.PostResult) bool {
	for _, r := range results[1:] {
		if r.Text != results[0].Text {
			return false
		}
	}
	return true
}

// PlatformLabel returns the human-friendly name for a platform. Exported
// because the social service uses it in per-platform failure titles too.
func PlatformLabel(p social.Platform) string {
	switch p {
	case social.Bluesky:
		return "Bluesky"
	case social.LinkedIn:
		return "LinkedIn"
	case social.Mastodon:
		return "Mastodon"
	default:
		return string(p)
	}
}

// kindLabel renders a PostKind as a human title, e.g. "new_source" ->
// "New source".
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
