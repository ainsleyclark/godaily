// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package slackdata builds the Slack Block Kit message for the engagement
// weekly roundup. It owns message layout and metric formatting only; the
// engagement service gathers the data and decides when to send. Its name
// avoids clashing with the pkg/gateway/slack send-channel.
package slackdata

import (
	"fmt"
	"strings"
	"time"

	engagement "github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// RoundupData is everything the weekly roundup card renders. PrevSummary is
// the prior window's headline stats, used only for delta arrows.
type RoundupData struct {
	From, To    time.Time
	Summary     engagement.SummaryStats
	PrevSummary engagement.SummaryStats
	Subs        engagement.SubscriberData
	Items       []engagement.ItemMetrics
	Tags        []engagement.TagMetrics
	Sources     []engagement.SourceMetrics
	BestIssue   *engagement.IssueEngagement
}

// Roundup builds the Friday weekly roundup card: scannable headline and
// subscriber KPIs with deltas, ranked top links, and the best issue with a
// link to its analytics.
func Roundup(d RoundupData) slack.Request {
	dateRange := fmt.Sprintf("%s – %s", d.From.Format("2 Jan"), d.To.Format("2 Jan"))

	blocks := []slack.Block{
		header("Weekly Roundup"),
		context("GoDaily  ·  " + dateRange),
		divider(),
		fields("*Headline*", headlineFields(d.Summary, d.PrevSummary)),
		context(fmt.Sprintf("Bounced %d  ·  Complained %d", d.Summary.Bounced, d.Summary.Complained)),
		subscriberBlock(d.Subs),
		divider(),
		section("*Top links*\n" + topLinkLines(d.Items)),
	}

	if extras := tagSourceLine(d.Tags, d.Sources); extras != "" {
		blocks = append(blocks, context(extras))
	}

	if d.BestIssue != nil {
		bi := d.BestIssue
		text := fmt.Sprintf("*Best issue:* %s  ·  %.1f%% click rate  ·  %.1f%% open rate",
			bi.Slug, bi.ClickRate*100, bi.OpenRate*100)
		blocks = append(blocks, sectionWithButton(text, "View analytics",
			fmt.Sprintf("%s/issues/%d", env.DashboardURL, bi.IssueID), ""))
	}

	blocks = append(blocks, context(fmt.Sprintf("<%s|Open the dashboard>", env.DashboardURL)))

	return message(fmt.Sprintf("GoDaily Weekly Roundup (%s)", dateRange), slack.ColorInfo, blocks)
}

// headlineFields returns the four headline KPIs as two-column section
// fields, each annotated with its delta versus the prior window.
func headlineFields(cs, ps engagement.SummaryStats) []string {
	return []string{
		fmt.Sprintf("*Issues sent*\n%d  %s", cs.IssuesSent, deltaCount(cs.IssuesSent, ps.IssuesSent)),
		fmt.Sprintf("*Delivered*\n%s  %s", humanCount(cs.Delivered), deltaCount(cs.Delivered, ps.Delivered)),
		fmt.Sprintf("*Open rate*\n%.1f%%  %s", cs.OpenRate*100, deltaPoint(cs.OpenRate, ps.OpenRate)),
		fmt.Sprintf("*Click rate*\n%.1f%%  %s", cs.ClickRate*100, deltaPoint(cs.ClickRate, ps.ClickRate)),
	}
}

// subscriberBlock renders the subscriber stats as a fields section, or a
// single line when there was no activity in the window.
func subscriberBlock(d engagement.SubscriberData) slack.Block {
	sp, ok := lastSubscriberPoint(d)
	if !ok {
		return section("*Subscribers*\nNo subscriber activity this week")
	}
	return fields("*Subscribers*", []string{
		fmt.Sprintf("*New*\n+%d", sp.New),
		fmt.Sprintf("*Net change*\n%s", signed(sp.NetChange)),
		fmt.Sprintf("*Confirmed*\n%d", sp.Confirmed),
		fmt.Sprintf("*Active*\n%s", humanCount(sp.ActiveAtEnd)),
	})
}

func topLinkLines(items []engagement.ItemMetrics) string {
	if len(items) == 0 {
		return "• No clicks recorded this week"
	}
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = fmt.Sprintf("%d. <%s|%s>  ·  %d clicks  ·  %s",
			i+1, it.URL, it.Title, it.Clicks, it.Source)
	}
	return strings.Join(parts, "\n")
}

func tagSourceLine(tags []engagement.TagMetrics, sources []engagement.SourceMetrics) string {
	var parts []string
	if len(tags) > 0 {
		ts := make([]string, len(tags))
		for i, t := range tags {
			ts[i] = fmt.Sprintf("%s (%d)", t.Tag, t.Clicks)
		}
		parts = append(parts, "*Top tags*: "+strings.Join(ts, " · "))
	}
	if len(sources) > 0 {
		ss := make([]string, len(sources))
		for i, src := range sources {
			ss[i] = fmt.Sprintf("%s (%d)", src.Source, src.Clicks)
		}
		parts = append(parts, "*Top sources*: "+strings.Join(ss, " · "))
	}
	return strings.Join(parts, "  ·  ")
}

// lastSubscriberPoint returns the most recent point in the series, if any.
func lastSubscriberPoint(d engagement.SubscriberData) (engagement.SubscriberPoint, bool) {
	if len(d.Points) == 0 {
		return engagement.SubscriberPoint{}, false
	}
	return d.Points[len(d.Points)-1], true
}

// deltaCount returns a directional percentage delta for two counts, e.g.
// "(↑ +12%)". When the prior value is zero, percentages are meaningless, so
// the absolute change is shown instead ("(new)" / "(–N)").
func deltaCount(curr, prev int64) string {
	if prev == 0 {
		switch {
		case curr == 0:
			return "(–)"
		case curr > 0:
			return "(new)"
		default:
			return fmt.Sprintf("(%s)", signed(curr))
		}
	}
	pct := (float64(curr) - float64(prev)) / float64(prev) * 100
	return formatDelta(pct, "%")
}

// deltaPoint returns a percentage-point delta for two rates expressed as
// 0..1 fractions, e.g. "(↑ +2.1pp)".
func deltaPoint(curr, prev float64) string {
	if curr == 0 && prev == 0 {
		return "(–)"
	}
	return formatDelta((curr-prev)*100, "pp")
}

// formatDelta renders a signed delta with an arrow and a unit suffix.
func formatDelta(v float64, unit string) string {
	switch {
	case v > 0:
		return fmt.Sprintf("(↑ +%.1f%s)", v, unit)
	case v < 0:
		return fmt.Sprintf("(↓ %.1f%s)", v, unit)
	default:
		return "(–)"
	}
}

// signed formats an int64 with an explicit + sign for non-negative values.
func signed(n int64) string {
	if n >= 0 {
		return fmt.Sprintf("+%d", n)
	}
	return fmt.Sprintf("%d", n)
}

// humanCount formats a count, using thousands separators for values under
// 10,000 and a compact "k" suffix above that.
func humanCount(n int64) string {
	if n < 10_000 {
		return addThousandsSep(n)
	}
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}

// addThousandsSep inserts commas into an integer's decimal representation.
func addThousandsSep(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	first := len(s) % 3
	if first > 0 {
		b.WriteString(s[:first])
		if len(s) > first {
			b.WriteString(",")
		}
	}
	for i := first; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteString(",")
		}
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
