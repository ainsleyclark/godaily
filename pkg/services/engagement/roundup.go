// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package engagement composes engagement queries into higher-level reports.
//
// Today it powers the Friday Slack roundup; it is the natural home for any
// future AI-driven analysis that needs a single struct of "everything we know
// about a window" without re-orchestrating the underlying queries.
package engagement

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	engagement "github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	slacksdk "github.com/slack-go/slack"
)

const (
	// topItemsLimit is the number of top-clicked links shown in the roundup.
	topItemsLimit = 5
	// topTagsLimit is the number of top tags shown in the roundup.
	topTagsLimit = 3
	// topSourcesLimit is the number of top sources shown in the roundup.
	topSourcesLimit = 3
)

// Snapshot is a single window's worth of engagement data, gathered once and
// reusable for formatting, comparison, or analysis.
type Snapshot struct {
	From, To  time.Time
	Summary   engagement.SummaryStats
	Subs      engagement.SubscriberData
	Items     []engagement.ItemMetrics
	Tags      []engagement.TagMetrics
	Sources   []engagement.SourceMetrics
	BestIssue *engagement.IssueEngagement
}

var _ engagement.MetricsService = (*Service)(nil)

// Service composes engagement queries into reports.
type Service struct {
	metrics engagement.MetricsRepository
	slack   slack.Sender
	now     func() time.Time
}

// New creates a Service backed by the given repository and Slack sender.
func New(metrics engagement.MetricsRepository, slack slack.Sender) *Service {
	return &Service{
		metrics: metrics,
		slack:   slack,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// Gather queries every relevant metric for the window [from, to] and returns
// it as a single Snapshot. Top-N lists are capped at the roundup's display
// limits; callers that need different limits should query the repository
// directly.
func (s *Service) Gather(ctx context.Context, from, to time.Time) (Snapshot, error) {
	filter := engagement.MetricsFilter{From: &from, To: &to}

	summary, err := s.metrics.Summary(ctx, filter)
	if err != nil {
		return Snapshot{}, errors.Wrap(err, "summary")
	}

	subs, err := s.metrics.SubscriberGrowth(ctx, filter, "week")
	if err != nil {
		return Snapshot{}, errors.Wrap(err, "subscriber growth")
	}

	itemFilter := filter
	itemFilter.Limit = topItemsLimit
	items, err := s.metrics.ItemList(ctx, itemFilter)
	if err != nil {
		return Snapshot{}, errors.Wrap(err, "item list")
	}

	tagFilter := filter
	tagFilter.Limit = topTagsLimit
	tags, err := s.metrics.TagList(ctx, tagFilter)
	if err != nil {
		return Snapshot{}, errors.Wrap(err, "tag list")
	}

	sourceFilter := filter
	sourceFilter.Limit = topSourcesLimit
	sources, err := s.metrics.SourceList(ctx, sourceFilter)
	if err != nil {
		return Snapshot{}, errors.Wrap(err, "source list")
	}

	bestFilter := filter
	bestFilter.Limit = 1
	bestList, err := s.metrics.IssueList(ctx, bestFilter, "click_rate")
	if err != nil {
		return Snapshot{}, errors.Wrap(err, "issue list")
	}
	var best *engagement.IssueEngagement
	if len(bestList) > 0 {
		best = &bestList[0]
	}

	return Snapshot{
		From:      from,
		To:        to,
		Summary:   summary,
		Subs:      subs,
		Items:     items,
		Tags:      tags,
		Sources:   sources,
		BestIssue: best,
	}, nil
}

// Roundup gathers the last 7 days of metrics plus the prior 7 days for
// comparison, formats them as a Slack message and sends it.
func (s *Service) Roundup(ctx context.Context) error {
	now := s.now()
	// Anchor to midnight UTC so digests sent earlier in the day (08:00 UTC)
	// are always inside the window regardless of when the cron fires.
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	currFrom := today.AddDate(0, 0, -7)
	prevFrom := currFrom.AddDate(0, 0, -7)

	curr, err := s.Gather(ctx, currFrom, now)
	if err != nil {
		return errors.Wrap(err, "gathering current window")
	}
	prev, err := s.Gather(ctx, prevFrom, currFrom)
	if err != nil {
		return errors.Wrap(err, "gathering prior window")
	}

	return s.slack.Send(ctx, roundupRequest(curr, prev))
}

// roundupRequest builds the Slack block-kit message from the current and
// prior snapshots. The prior snapshot is used only for delta arrows.
func roundupRequest(curr, prev Snapshot) slack.Request {
	dateRange := fmt.Sprintf("%s – %s",
		curr.From.Format("2 Jan"), curr.To.Format("2 Jan"))

	blocks := []slack.Block{
		slacksdk.NewHeaderBlock(plain("GoDaily — Weekly Roundup")),
		slacksdk.NewContextBlock("", plain(dateRange)),
		slacksdk.NewDividerBlock(),
		section("*Headline*\n" + headlineLines(curr.Summary, prev.Summary)),
		section("*Subscribers*\n" + subscriberLines(curr.Subs)),
		section("*Top links*\n" + topLinkLines(curr.Items)),
	}

	if extras := tagSourceLine(curr); extras != "" {
		blocks = append(blocks, slacksdk.NewContextBlock("", mrkdwn(extras)))
	}

	if curr.BestIssue != nil {
		bi := curr.BestIssue
		blocks = append(blocks, section(fmt.Sprintf(
			"*Best issue*: %s — %.1f%% click rate, %.1f%% open rate",
			bi.Slug, bi.ClickRate*100, bi.OpenRate*100,
		)))
	}

	return slack.Request{
		Text:        fmt.Sprintf("GoDaily — Weekly Roundup (%s)", dateRange),
		Blocks:      slack.BlockSet{BlockSet: blocks},
		Attachments: []slack.Attachment{{Color: slack.ColorInfo}},
	}
}

func headlineLines(cs, ps engagement.SummaryStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "• Issues sent: %d  %s\n",
		cs.IssuesSent, deltaCount(cs.IssuesSent, ps.IssuesSent))
	fmt.Fprintf(&b, "• Delivered: %s  %s\n",
		humanCount(cs.Delivered), deltaCount(cs.Delivered, ps.Delivered))
	fmt.Fprintf(&b, "• Opens: %s unique / %.1f%% open rate  %s\n",
		humanCount(cs.UniqueOpens), cs.OpenRate*100, deltaPoint(cs.OpenRate, ps.OpenRate))
	fmt.Fprintf(&b, "• Clicks: %s unique / %.1f%% click rate  %s\n",
		humanCount(cs.UniqueClicks), cs.ClickRate*100, deltaPoint(cs.ClickRate, ps.ClickRate))
	fmt.Fprintf(&b, "• Bounced %d · Complained %d", cs.Bounced, cs.Complained)
	return b.String()
}

func subscriberLines(d engagement.SubscriberData) string {
	sp, ok := lastSubscriberPoint(d)
	if !ok {
		return "• No subscriber activity this week"
	}
	return fmt.Sprintf(
		"• +%d new, %d confirmed, %d unsubscribed → net %s\n• Active: %s",
		sp.New, sp.Confirmed, sp.Unsubscribed, signed(sp.NetChange), humanCount(sp.ActiveAtEnd),
	)
}

func topLinkLines(items []engagement.ItemMetrics) string {
	if len(items) == 0 {
		return "• No clicks recorded this week"
	}
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = fmt.Sprintf("%d. <%s|%s> — %d clicks · %s",
			i+1, it.URL, it.Title, it.Clicks, it.Source)
	}
	return strings.Join(parts, "\n")
}

func tagSourceLine(curr Snapshot) string {
	var parts []string
	if len(curr.Tags) > 0 {
		tags := make([]string, len(curr.Tags))
		for i, t := range curr.Tags {
			tags[i] = fmt.Sprintf("%s (%d)", t.Tag, t.Clicks)
		}
		parts = append(parts, "*Top tags*: "+strings.Join(tags, " · "))
	}
	if len(curr.Sources) > 0 {
		srcs := make([]string, len(curr.Sources))
		for i, src := range curr.Sources {
			srcs[i] = fmt.Sprintf("%s (%d)", src.Source, src.Clicks)
		}
		parts = append(parts, "*Top sources*: "+strings.Join(srcs, " · "))
	}
	return strings.Join(parts, "  ·  ")
}

func plain(text string) *slack.TextObject {
	return slacksdk.NewTextBlockObject(slacksdk.PlainTextType, text, false, false)
}

func mrkdwn(text string) *slack.TextObject {
	return slacksdk.NewTextBlockObject(slacksdk.MarkdownType, text, false, false)
}

func section(text string) *slack.Section {
	return slacksdk.NewSectionBlock(mrkdwn(text), nil, nil)
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
