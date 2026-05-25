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
	"log/slog"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// CollectOptions configures a Collect call.
type CollectOptions struct {
	// DryRun skips persisting items; only the raw source items are returned.
	DryRun bool

	// Sources restricts the run to the given sources. If empty,
	// all registered sources (news.Sources) are used.
	Sources []news.Source
}

// CollectResponse is the result of a Collect call. Sources contains the
// fetched items grouped by source. Errors contains a per-source error for any
// source that failed to fetch; a source absent from Errors succeeded (even if
// it returned zero items, which is normal on quiet days).
type CollectResponse struct {
	Sources []news.SourceItems
	Errors  map[news.Source]error
}

// Collect fetches Go news items from all registered sources within the current
// collection window, scores and sorts them, and (unless DryRun) persists them
// as unlinked items in the database (issue_id = nil).
func (a Aggregator) Collect(ctx context.Context, opts CollectOptions) (CollectResponse, error) {
	start, end := collectWindow(time.Now())

	sources := opts.Sources
	if len(sources) == 0 {
		sources = news.Sources
	}

	if !opts.DryRun && a.items != nil {
		existing, err := a.items.List(ctx, news.ItemListOptions{From: &start, To: &end})
		if err != nil {
			return CollectResponse{}, errors.Wrap(err, "checking existing items")
		}
		if len(existing) > 0 {
			slog.InfoContext(ctx, "Items already collected for window, skipping", "start", start.Format("2006-01-02"), "end", end.Format("2006-01-02"), "count", len(existing))
			return CollectResponse{}, nil
		}
	}

	var (
		results    []news.SourceItems
		sourceErrs map[news.Source]error
	)
	for _, src := range sources {
		fetched, err := a.fetchSource(ctx, src)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to fetch source", "source", src, "err", err)
			if sourceErrs == nil {
				sourceErrs = make(map[news.Source]error)
			}
			sourceErrs[src] = err
			continue
		}
		si := news.SourceItems{Source: src}

		for _, item := range fetched {
			if item.Published.IsZero() {
				slog.ErrorContext(ctx, "Item has zero published date", "source", src, "title", item.Title)
				continue
			}
			// Sources that set Published: time.Now() (e.g. meetup) produce a
			// timestamp that is always >= end (today midnight). Clamp those to
			// start+1h so they land inside this window without the source needing
			// to know anything about the pipeline's date expectations.
			if !item.Published.Before(end) {
				item.Published = start.Add(time.Hour)
			}
			if item.Published.After(start) {
				si.Items = append(si.Items, item)
			}
		}

		slog.InfoContext(ctx, "Date-filtered source", "source", src, "kept", len(si.Items), "total", len(fetched))
		if len(si.Items) > 0 {
			sort.SliceStable(si.Items, func(i, j int) bool {
				return si.Items[i].Score > si.Items[j].Score
			})
			results = append(results, si)
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Source.Priority() > results[j].Source.Priority()
	})

	resp := CollectResponse{Sources: results, Errors: sourceErrs}

	if opts.DryRun || len(results) == 0 {
		if !opts.DryRun && a.items != nil {
			slog.WarnContext(ctx, "No items found for date window", "start", start.Format("2006-01-02"))
		}
		return resp, nil
	}

	var position int
	for _, section := range results {
		for _, item := range section.Items {
			position++
			item.Source = section.Source
			if _, err := a.items.Create(ctx, nil, position, item); err != nil {
				return resp, errors.Wrap(err, "creating news item")
			}
		}
	}

	slog.InfoContext(ctx, "Collected items", "start", start.Format("2006-01-02"), "end", end.Format("2006-01-02"), "count", position)

	return resp, nil
}

// collectWindow returns the date range to collect for a given time. The window
// is always yesterday-to-today (one day) to capture items published yesterday.
func collectWindow(now time.Time) (start, end time.Time) {
	today := now.UTC().Truncate(24 * time.Hour)
	return today.AddDate(0, 0, -1), today
}
