// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
)

// Collect fetches Go news items from all registered sources within the current
// collection window, scores and sorts them, and (unless DryRun) persists them
// as unlinked items in the database (issue_id = nil).
func (s Service) Collect(ctx context.Context, opts digest.CollectOptions) (digest.CollectResponse, error) {
	start, end := collectWindow(time.Now())

	sources := opts.Sources
	if len(sources) == 0 {
		sources = news.Sources
	}

	if !opts.DryRun && s.items != nil {
		existing, err := s.items.List(ctx, news.ItemListOptions{From: &start, To: &end})
		if err != nil {
			return digest.CollectResponse{}, errors.Wrap(err, "checking existing items")
		}
		if len(existing) > 0 {
			slog.InfoContext(ctx, "Items already collected for window, skipping", "start", start.Format("2006-01-02"), "end", end.Format("2006-01-02"), "count", len(existing))
			return digest.CollectResponse{}, nil
		}
	}

	var (
		results    []news.SourceItems
		sourceErrs map[news.Source]error
	)
	for _, src := range sources {
		fetched, err := s.fetchSource(ctx, src)
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
			}
			if clamped, ok := windowClamp(item, start, end); ok {
				si.Items = append(si.Items, clamped)
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

	resp := digest.CollectResponse{Sources: results, Errors: sourceErrs}

	if opts.DryRun || len(results) == 0 {
		if !opts.DryRun && s.items != nil {
			slog.WarnContext(ctx, "No items found for date window", "start", start.Format("2006-01-02"))
		}
		return resp, nil
	}

	var position int
	for _, section := range results {
		for _, item := range section.Items {
			position++
			item.Source = section.Source
			if _, err := s.items.Create(ctx, nil, position, item); err != nil {
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

// windowClamp adjusts an item's Published date to fit the collection window and
// reports whether it should be kept. Items with a zero published date are
// dropped. Future-dated items (Published >= end, e.g. sources that stamp
// time.Now()) are clamped to start+1h so they land inside the window without
// the source needing to know anything about the pipeline's date expectations.
// Items published at or before start are dropped as out-of-window.
func windowClamp(item news.Item, start, end time.Time) (news.Item, bool) {
	if item.Published.IsZero() {
		return item, false
	}
	if !item.Published.Before(end) {
		item.Published = start.Add(time.Hour)
	}
	return item, item.Published.After(start)
}
