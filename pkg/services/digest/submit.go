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

// Submit persists a manually supplied set of items for a single source,
// applying the same collection-window filtering, future-date clamping, and
// score ordering as Collect. It is the fallback for sources whose live fetch is
// blocked (e.g. Reddit via ScraperAPI): an operator pastes the raw listing and
// the items land in the current window exactly as an automated collection would
// have placed them.
//
// Unlike Collect — which skips the whole run if any items already exist for the
// window — Submit only skips when items for this specific source already exist.
// That makes re-submitting idempotent while allowing a source to be back-filled
// after the rest of the run succeeded.
func (s Service) Submit(ctx context.Context, source news.Source, items []news.Item) (digest.SubmitResponse, error) {
	start, end := collectWindow(time.Now())
	resp := digest.SubmitResponse{Received: len(items)}

	if s.items != nil {
		existing, err := s.items.List(ctx, news.ItemListOptions{
			From:    &start,
			To:      &end,
			Sources: []news.Source{source},
		})
		if err != nil {
			return resp, errors.Wrap(err, "checking existing items")
		}
		if len(existing) > 0 {
			slog.InfoContext(ctx, "Source already has items for window, skipping submission",
				"source", source, "count", len(existing), "start", start.Format("2006-01-02"))
			resp.Skipped = true
			return resp, nil
		}
	}

	var kept []news.Item
	for _, item := range items {
		item.Source = source
		if clamped, ok := windowClamp(item, start, end); ok {
			kept = append(kept, clamped)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool {
		return kept[i].Score > kept[j].Score
	})

	if s.items == nil {
		resp.Persisted = len(kept)
		return resp, nil
	}

	var position int
	for _, item := range kept {
		position++
		if _, err := s.items.Create(ctx, nil, position, item); err != nil {
			return resp, errors.Wrap(err, "creating news item")
		}
	}
	resp.Persisted = position

	slog.InfoContext(ctx, "Submitted items", "source", source,
		"received", resp.Received, "persisted", resp.Persisted, "start", start.Format("2006-01-02"))

	return resp, nil
}
