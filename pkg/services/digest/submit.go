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
// blocked (e.g. Reddit via ScraperAPI): an operator (or a scheduled script)
// posts the raw listing and the items land in the current window exactly as an
// automated collection would have placed them.
//
// Submit de-duplicates proactively on (url, tag) — the same key as the
// items_url_tag_unique index — against both items already stored in the window
// and repeats within the payload itself. This means it can be run repeatedly
// (e.g. on a schedule) without creating duplicates and without relying on the
// database constraint to reject them. Every item that survives windowClamp
// falls inside [start, end], which is exactly the range checked for existing
// items, so no duplicate can reach the store via this path.
func (s Service) Submit(ctx context.Context, source news.Source, items []news.Item) (digest.SubmitResponse, error) {
	start, end := collectWindow(time.Now())
	resp := digest.SubmitResponse{Received: len(items)}

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

	existing, err := s.items.List(ctx, news.ItemListOptions{From: &start, To: &end})
	if err != nil {
		return resp, errors.Wrap(err, "checking existing items")
	}
	seen := make(map[string]bool, len(existing)+len(kept))
	for _, e := range existing {
		seen[dedupKey(e)] = true
	}

	// Continue positions after the items already in the window so manual
	// submissions append rather than collide with an earlier collection.
	position := len(existing)
	for _, item := range kept {
		key := dedupKey(item)
		if seen[key] {
			resp.Duplicates++
			continue
		}
		seen[key] = true
		position++
		if _, err := s.items.Create(ctx, nil, position, item); err != nil {
			return resp, errors.Wrap(err, "creating news item")
		}
		resp.Persisted++
	}

	slog.InfoContext(ctx, "Submitted items", "source", source,
		"received", resp.Received, "persisted", resp.Persisted,
		"duplicates", resp.Duplicates, "start", start.Format("2006-01-02"))

	return resp, nil
}

// dedupKey returns the uniqueness key for an item, matching the
// items_url_tag_unique database index on (url, tag).
func dedupKey(i news.Item) string {
	return i.URL + "\x00" + string(i.Tag)
}
