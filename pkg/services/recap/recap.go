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

// Package recap computes "top items in the last N days" from email click
// engagement. It is the single source of truth for the weekly recap
// dataset and is consumed by the social rotation, the email outro, the
// /this-week web page, and an eventual RSS feed.
//
// The package has no dependency on the social stack — it returns a
// structured Top value and lets callers decide how to render it.
package recap

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

// defaultLimit caps the recap shortlist size when the caller doesn't
// specify one. Three items fits cleanly into a social post; consumers
// that want more (e.g. an email outro) can override.
const defaultLimit = 3

type (
	// Period describes the [Start, End) window covered by a recap.
	Period struct {
		Start time.Time
		End   time.Time
		// Label is an ISO-week style identifier used for idempotency
		// keys, e.g. "2026-W21". Stable across reruns within the same
		// week.
		Label string
	}
	// RankedItem is one entry in a recap, paired with its click count.
	RankedItem struct {
		engagement.ItemMetrics
	}
	// Top is the recap dataset.
	Top struct {
		Period Period
		Items  []RankedItem
	}
	// TopOptions tunes a Top call.
	TopOptions struct {
		// N caps the returned items. Zero means defaultLimit (3).
		N int
		// Window is the lookback duration ending at "now". Zero means
		// "since Monday 00:00 UTC of now's week" — the natural Mon→Fri
		// recap window.
		Window time.Duration
		// MinItems is the floor below which Top returns its zero value
		// (so the caller can no-op cleanly). Defaults to 0 — every
		// result kept.
		MinItems int
	}
)

// HasItems reports whether the recap has any ranked items at all.
func (t Top) HasItems() bool { return len(t.Items) > 0 }

// Service computes recap datasets from the metrics repository.
type Service struct {
	metrics engagement.MetricsRepository
}

// New returns a Service backed by the given metrics repository.
func New(metrics engagement.MetricsRepository) (*Service, error) {
	if metrics == nil {
		return nil, errors.New("recap: metrics repository is required")
	}
	return &Service{metrics: metrics}, nil
}

// Top returns the most-clicked items in the window ending at now. When
// the dataset has fewer than opts.MinItems entries, it returns the zero
// value (and an empty period) — the caller treats that as "skip".
func (s *Service) Top(ctx context.Context, now time.Time, opts TopOptions) (Top, error) {
	limit := opts.N
	if limit <= 0 {
		limit = defaultLimit
	}

	period := makePeriod(now.UTC(), opts.Window)

	items, err := s.metrics.ItemList(ctx, engagement.MetricsFilter{
		From:  &period.Start,
		To:    &period.End,
		Limit: limit,
	})
	if err != nil {
		return Top{}, errors.Wrap(err, "metrics.ItemList")
	}

	if opts.MinItems > 0 && len(items) < opts.MinItems {
		return Top{}, nil
	}

	ranked := make([]RankedItem, 0, len(items))
	for _, it := range items {
		ranked = append(ranked, RankedItem{ItemMetrics: it})
	}
	return Top{Period: period, Items: ranked}, nil
}

// makePeriod builds the [Start, End) window. When window is zero the
// start is the Monday 00:00 UTC of now's ISO week, so a Friday call
// returns Mon-Thu activity (and a Thursday call returns Mon-Wed).
func makePeriod(now time.Time, window time.Duration) Period {
	now = now.UTC()
	var start time.Time
	if window > 0 {
		start = now.Add(-window)
	} else {
		start = mondayOf(now)
	}
	year, week := start.ISOWeek()
	return Period{
		Start: start,
		End:   now,
		Label: fmt.Sprintf("%d-W%02d", year, week),
	}
}

// mondayOf returns 00:00 UTC on the Monday of t's ISO week.
func mondayOf(t time.Time) time.Time {
	t = t.UTC()
	wd := int(t.Weekday()) // Sun=0 .. Sat=6
	if wd == 0 {
		wd = 7 // treat Sunday as end-of-previous-ISO-week
	}
	offset := time.Duration(wd-1) * 24 * time.Hour
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return day.Add(-offset)
}
