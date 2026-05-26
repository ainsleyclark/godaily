// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
	"github.com/ainsleyclark/godaily/pkg/store"
	"github.com/pkg/errors"
)

// Post is the entry point for the daily featured slot. It:
//  1. Loads the issue for Date (returns store.ErrNotFound if there is none).
//  2. Loads the issue's items.
//  3. Picks the day's featured item via the AI prompter.
//  4. For each enabled poster: skips if already posted today; otherwise
//     reframes the featured item for that platform, publishes it, and
//     records the result.
func (s *Service) Post(ctx context.Context, opts social.PostOptions) ([]social.PostResult, error) {
	if len(s.posters) == 0 {
		slog.InfoContext(ctx, "Skipping social — no posters configured")
		return nil, nil
	}

	date := opts.Date.UTC()
	slug := date.Format("2006-01-02")

	issue, err := s.issues.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("no digest found for %s — run `godaily collect` first", slug)
		}
		return nil, errors.Wrap(err, "loading digest")
	}

	rows, err := s.items.List(ctx, news.ItemListOptions{IssueID: &issue.ID})
	if err != nil {
		return nil, errors.Wrap(err, "loading items")
	}
	if len(rows) == 0 {
		slog.InfoContext(ctx, "Skipping social — no items for issue", "issue", issue.ID)
		return nil, nil
	}

	feat, err := featured.Feature(ctx, s.prompter, date, rows)
	if err != nil {
		s.notifyFailure(ctx, "AI feature pick failed: "+err.Error())
		return nil, errors.Wrap(err, "feature")
	}

	slog.InfoContext(
		ctx, "Selected featured item",
		"title", feat.Title, "url", feat.URL, "tag", string(feat.Tag),
	)

	wanted := selectPosters(s.posters, opts.Platforms)
	issueID := issue.ID

	return s.publish(ctx, publishCtx{
		platforms: wanted,
		dryRun:    opts.DryRun,
		kind:      social.PostKindFeatured,
		issueID:   &issueID,
		generate: func(_ context.Context, p social.Platform) (string, error) {
			reframe, ok := s.reframers[p]
			if !ok {
				return "", fmt.Errorf("no reframer registered for platform %s", p)
			}
			return reframe(ctx, s.prompter, feat)
		},
		// Featured posts use (issue_id, platform) for idempotency.
		skipIfPosted: func(ctx context.Context, p string) (bool, error) {
			return s.posts.HasPosted(ctx, issueID, p)
		},
	})
}
