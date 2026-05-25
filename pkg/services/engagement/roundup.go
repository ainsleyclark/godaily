// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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

	domengagement "github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
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
	Summary   domengagement.SummaryStats
	Subs      domengagement.SubscriberData
	Items     []domengagement.ItemMetrics
	Tags      []domengagement.TagMetrics
	Sources   []domengagement.SourceMetrics
	BestIssue *domengagement.IssueEngagement
}

// Service composes engagement queries into reports.
type Service struct {
	metrics domengagement.MetricsRepository
	slack   slack.Sender
	now     func() time.Time
}

// New creates a Service backed by the given repository and Slack sender.
func New(metrics domengagement.MetricsRepository, slack slack.Sender) *Service {
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
	filter := domengagement.MetricsFilter{From: &from, To: &to}

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
	var best *domengagement.IssueEngagement
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

	return s.slack.Send(ctx, formatRoundup(curr, prev))
}

// formatRoundup builds the Slack mrkdwn message from the current and prior
// snapshots. The prior snapshot is used only for delta arrows.
func formatRoundup(curr, prev Snapshot) string {
	var b strings.Builder
	dateRange := fmt.Sprintf("%s – %s",
		curr.From.Format("2 Jan"), curr.To.Format("2 Jan"))

	fmt.Fprintf(&b, "*GoDaily — Weekly Roundup* (%s)\n\n", dateRange)

	// Headline.
	cs, ps := curr.Summary, prev.Summary
	b.WriteString("*Headline*\n")
	fmt.Fprintf(&b, "• Issues sent: %d  %s\n",
		cs.IssuesSent, deltaCount(cs.IssuesSent, ps.IssuesSent))
	fmt.Fprintf(&b, "• Delivered: %s  %s\n",
		humanCount(cs.Delivered), deltaCount(cs.Delivered, ps.Delivered))
	fmt.Fprintf(&b, "• Opens: %s unique / %.1f%% open rate  %s\n",
		humanCount(cs.UniqueOpens), cs.OpenRate*100, deltaPoint(cs.OpenRate, ps.OpenRate))
	fmt.Fprintf(&b, "• Clicks: %s unique / %.1f%% click rate  %s\n",
		humanCount(cs.UniqueClicks), cs.ClickRate*100, deltaPoint(cs.ClickRate, ps.ClickRate))
	fmt.Fprintf(&b, "• Bounced %d · Complained %d\n\n", cs.Bounced, cs.Complained)

	// Subscribers.
	b.WriteString("*Subscribers*\n")
	if sp, ok := lastSubscriberPoint(curr.Subs); ok {
		fmt.Fprintf(&b, "• +%d new, %d confirmed, %d unsubscribed → net %s\n",
			sp.New, sp.Confirmed, sp.Unsubscribed, signed(sp.NetChange))
		fmt.Fprintf(&b, "• Active: %s\n\n", humanCount(sp.ActiveAtEnd))
	} else {
		b.WriteString("• No subscriber activity this week\n\n")
	}

	// Top links.
	b.WriteString("*Top links*\n")
	if len(curr.Items) == 0 {
		b.WriteString("• No clicks recorded this week\n\n")
	} else {
		for i, it := range curr.Items {
			fmt.Fprintf(&b, "%d. <%s|%s> — %d clicks · %s\n",
				i+1, it.URL, it.Title, it.Clicks, it.Source)
		}
		b.WriteString("\n")
	}

	// Top tags.
	if len(curr.Tags) > 0 {
		b.WriteString("*Top tags*: ")
		parts := make([]string, len(curr.Tags))
		for i, t := range curr.Tags {
			parts[i] = fmt.Sprintf("%s (%d)", t.Tag, t.Clicks)
		}
		b.WriteString(strings.Join(parts, " · "))
		b.WriteString("\n")
	}

	// Top sources.
	if len(curr.Sources) > 0 {
		b.WriteString("*Top sources*: ")
		parts := make([]string, len(curr.Sources))
		for i, src := range curr.Sources {
			parts[i] = fmt.Sprintf("%s (%d)", src.Source, src.Clicks)
		}
		b.WriteString(strings.Join(parts, " · "))
		b.WriteString("\n")
	}

	// Best issue.
	if curr.BestIssue != nil {
		bi := curr.BestIssue
		fmt.Fprintf(&b, "\n*Best issue*: %s — %.1f%% click rate, %.1f%% open rate\n",
			bi.Slug, bi.ClickRate*100, bi.OpenRate*100)
	}

	return strings.TrimRight(b.String(), "\n")
}

// lastSubscriberPoint returns the most recent point in the series, if any.
func lastSubscriberPoint(d domengagement.SubscriberData) (domengagement.SubscriberPoint, bool) {
	if len(d.Points) == 0 {
		return domengagement.SubscriberPoint{}, false
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
