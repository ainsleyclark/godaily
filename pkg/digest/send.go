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
	"log/slog"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// SendDigest loads the draft digest for the given date, sends it to the
// admin address and all active subscribers, then updates the stored issue status.
func (a Aggregator) SendDigest(ctx context.Context, date time.Time, force bool) error {
	slug := date.Format("2006-01-02")

	slog.InfoContext(ctx, "Preparing to send digest", "slug", slug)

	issue, err := a.issues.FindBySlug(ctx, slug)
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("no digest found for %s — run `godaily collect` first", slug)
	} else if err != nil {
		return errors.Wrap(err, "loading digest")
	} else if !force && issue.Status != news.IssueStatusDraft {
		return fmt.Errorf("digest for %s has status %q, expected %q", slug, issue.Status, news.IssueStatusDraft)
	}

	sections, err := loadSections(ctx, a.items, issue.ID)
	if err != nil {
		return errors.Wrap(err, "loading sections")
	}

	subs, err := a.subscribers.ListActive(ctx)
	if err != nil {
		return errors.Wrap(err, "listing active subscribers")
	}

	canonicalURL := env.AppURL + "/issues/" + issue.Slug + "/"

	// Render and send to admin (no unsubscribe link).
	adminRendered, err := renderDigest(digestOptions{Day: date, Sources: sections, CanonicalURL: canonicalURL})
	if err != nil {
		return errors.Wrap(err, "rendering digest")
	}

	status := news.IssueStatusSent
	if err = a.sendRendered(ctx, a.adminEmailAddress, adminRendered); err != nil {
		slog.ErrorContext(ctx, "Failed to send digest email to admin", "err", err)
		status = news.IssueStatusError
	}

	// Send personalized digests to active subscribers.
	for _, sub := range subs {
		if sub.UnsubscribeToken == "" {
			slog.ErrorContext(ctx, "Skipping subscriber with missing unsubscribe token", "email", sub.Email)
			continue
		}
		unsubURL := env.AppURL + "/api/unsubscribe?token=" + sub.UnsubscribeToken
		subRendered, renderErr := renderDigest(digestOptions{Day: date, Sources: sections, UnsubscribeURL: unsubURL, CanonicalURL: canonicalURL})
		if renderErr != nil {
			slog.ErrorContext(ctx, "Failed to render digest for subscriber", "email", sub.Email, "err", renderErr)
			continue
		}
		if sendErr := a.sendRendered(ctx, sub.Email, subRendered); sendErr != nil {
			slog.ErrorContext(ctx, "Failed to send digest to subscriber", "email", sub.Email, "err", sendErr)
		}
	}

	if _, err = a.issues.UpdateStatus(ctx, issue.ID, status, time.Now().UTC()); err != nil {
		slog.ErrorContext(ctx, "Failed to update issue status", "err", err)
	}

	return nil
}

// loadSections fetches stored items for an issue and groups them into
// SourceItems slices sorted by source priority, matching the shape
// produced by Collect.
func loadSections(ctx context.Context, repo news.ItemRepository, issueID int64) ([]news.SourceItems, error) {
	items, err := repo.ListByIssue(ctx, issueID)
	if err != nil {
		return nil, err
	}

	order := make([]news.Source, 0)
	bySource := make(map[news.Source]*news.SourceItems)
	for _, item := range items {
		if _, ok := bySource[item.Source]; !ok {
			bySource[item.Source] = &news.SourceItems{Source: item.Source}
			order = append(order, item.Source)
		}
		bySource[item.Source].Items = append(bySource[item.Source].Items, item)
	}

	sections := make([]news.SourceItems, 0, len(bySource))
	for _, src := range order {
		sections = append(sections, *bySource[src])
	}

	sort.SliceStable(sections, func(i, j int) bool {
		return sections[i].Source.Priority() > sections[j].Source.Priority()
	})

	return sections, nil
}
