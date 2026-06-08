// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
)

// defaultLimit caps the recap shortlist size when the caller doesn't
// specify one. Three items fits cleanly into a social post; consumers
// that want more (e.g. an email outro) can override.
const defaultLimit = 3

// RecapService computes recap datasets from the metrics repository.
type RecapService struct {
	metrics engagement.MetricsRepository
}

// NewRecapService returns a RecapService backed by the given metrics repository.
func NewRecapService(metrics engagement.MetricsRepository) (*RecapService, error) {
	if metrics == nil {
		return nil, errors.New("recap: metrics repository is required")
	}
	return &RecapService{metrics: metrics}, nil
}

// Top returns the most-clicked items for the recap window. With the
// default (zero) window that is the previous complete ISO week; a
// non-zero window rolls back from now. When the dataset has fewer than
// opts.MinItems entries, it returns the zero value (and an empty
// period) — the caller treats that as "skip".
func (s *RecapService) Top(ctx context.Context, now time.Time, opts digest.TopOptions) (digest.Top, error) {
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
		return digest.Top{}, errors.Wrap(err, "metrics.ItemList")
	}

	if opts.MinItems > 0 && len(items) < opts.MinItems {
		return digest.Top{}, nil
	}

	ranked := make([]digest.RankedItem, 0, len(items))
	for _, it := range items {
		ranked = append(ranked, digest.RankedItem{ItemMetrics: it})
	}
	return digest.Top{Period: period, Items: ranked}, nil
}

// makePeriod builds the [Start, End) window. When window is zero the
// period is the previous complete ISO week: Monday 00:00 UTC of last
// week up to (but not including) Monday 00:00 UTC of now's week. The
// recap runs on Monday, so the default window is the seven days that
// just finished — Mon–Sun of last week — not the near-empty slice of
// the week now beginning. A non-zero window keeps the simple rolling
// "now minus window" semantics.
func makePeriod(now time.Time, window time.Duration) digest.Period {
	now = now.UTC()
	var start, end time.Time
	if window > 0 {
		start = now.Add(-window)
		end = now
	} else {
		end = mondayOf(now)
		start = end.Add(-7 * 24 * time.Hour)
	}
	year, week := start.ISOWeek()
	return digest.Period{
		Start: start,
		End:   end,
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
