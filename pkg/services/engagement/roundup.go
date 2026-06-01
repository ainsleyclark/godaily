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
	"time"

	"github.com/pkg/errors"

	engagement "github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/engagement/internal/slackdata"
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

	return s.slack.Send(ctx, slackdata.Roundup(slackdata.RoundupData{
		From:        curr.From,
		To:          curr.To,
		Summary:     curr.Summary,
		PrevSummary: prev.Summary,
		Subs:        curr.Subs,
		Items:       curr.Items,
		Tags:        curr.Tags,
		Sources:     curr.Sources,
		BestIssue:   curr.BestIssue,
	}))
}
