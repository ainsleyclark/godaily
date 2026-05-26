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

package digest

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	domaindigest "github.com/ainsleyclark/godaily/pkg/domain/digest"
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

// Top returns the most-clicked items in the window ending at now. When
// the dataset has fewer than opts.MinItems entries, it returns the zero
// value (and an empty period) — the caller treats that as "skip".
func (s *RecapService) Top(ctx context.Context, now time.Time, opts domaindigest.TopOptions) (domaindigest.Top, error) {
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
		return domaindigest.Top{}, errors.Wrap(err, "metrics.ItemList")
	}

	if opts.MinItems > 0 && len(items) < opts.MinItems {
		return domaindigest.Top{}, nil
	}

	ranked := make([]domaindigest.RankedItem, 0, len(items))
	for _, it := range items {
		ranked = append(ranked, domaindigest.RankedItem{ItemMetrics: it})
	}
	return domaindigest.Top{Period: period, Items: ranked}, nil
}

// makePeriod builds the [Start, End) window. When window is zero the
// start is the Monday 00:00 UTC of now's ISO week, so a Friday call
// returns Mon-Thu activity (and a Thursday call returns Mon-Wed).
func makePeriod(now time.Time, window time.Duration) domaindigest.Period {
	now = now.UTC()
	var start time.Time
	if window > 0 {
		start = now.Add(-window)
	} else {
		start = mondayOf(now)
	}
	year, week := start.ISOWeek()
	return domaindigest.Period{
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
