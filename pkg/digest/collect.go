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

	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// CollectOptions configures a Collect call.
type CollectOptions struct {
	// DryRun skips rendering and persisting the digest; only the raw
	// source items are returned.
	DryRun bool

	// Sources restricts the run to the given sources. If empty,
	// all registered sources (news.Sources) are used.
	Sources []news.Source
}

// Collect fetches Go news items from all registered sources within the current
// collection window, scores and sorts them, renders the digest and (unless
// DryRun) persists it as a draft issue in the database.
func (a Aggregator) Collect(ctx context.Context, opts CollectOptions) ([]news.SourceItems, error) {
	day, next := collectWindow(time.Now())

	sources := opts.Sources
	if len(sources) == 0 {
		sources = news.Sources
	}

	var results []news.SourceItems
	for _, src := range sources {
		fetched, err := a.fetchSource(ctx, src)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to fetch source", "source", src, "err", err)
			continue
		}
		si := news.SourceItems{Source: src}

		for _, item := range fetched {
			if item.Published.IsZero() {
				slog.ErrorContext(ctx, "Item has zero published date", "source", src, "title", item.Title)
				continue
			}
			if item.Published.After(day) && item.Published.Before(next) {
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

	if opts.DryRun || len(results) == 0 {
		if !opts.DryRun {
			slog.WarnContext(ctx, "No items found for date window, issue will not be created", "date", day.Format("2006-01-02"))
		}
		return results, nil
	}

	if _, err := renderDigest(digestOptions{Day: day, Sources: results}); err != nil {
		slog.ErrorContext(ctx, "Failed to render digest", "err", err)
		return results, nil
	}

	subject, summary := a.synthesiseDigestMeta(ctx, day, results)

	issue := news.Issue{
		Slug:    day.Format("2006-01-02"),
		Subject: subject,
		Summary: summary,
		Status:  news.IssueStatusDraft,
		SentAt:  time.Now().UTC(),
	}

	return results, a.persistIssue(ctx, issue, results)
}

// collectWindow returns the date range to collect for a given time. On Monday
// UTC the window covers Saturday and Sunday; on any other day it covers
// yesterday only.
func collectWindow(now time.Time) (start, end time.Time) {
	today := now.UTC().Truncate(24 * time.Hour)
	if now.UTC().Weekday() == time.Monday {
		return today.AddDate(0, 0, -2), today
	}
	return today.AddDate(0, 0, -1), today
}

func (a Aggregator) persistIssue(ctx context.Context, issue news.Issue, sections []news.SourceItems) error {
	_, err := a.issues.FindBySlug(ctx, issue.Slug)
	switch {
	case err == nil: // No error indicates it exists.
		slog.WarnContext(ctx, "Issue already persisted in the store, skipping", "slug", issue.Slug)
		return nil
	case !errors.Is(err, store.ErrNotFound): // Is a database error.
		return errors.Wrap(err, "checking existing issue")
	}

	created, err := a.issues.Create(ctx, issue)
	if err != nil {
		return errors.Wrap(err, "creating issue")
	}

	var position int
	for _, section := range sections {
		for _, item := range section.Items {
			position++
			item.Source = section.Source
			if _, err = a.items.Create(ctx, created.ID, position, item); err != nil {
				return errors.Wrap(err, "creating news item")
			}
		}
	}

	slog.InfoContext(ctx, "Persisted issue", "slug", issue.Slug)

	return nil
}

// synthesiseDigestMeta calls the synthesiser to generate the email subject title
// and intro paragraph. On failure it logs a warning and returns static fallbacks
// so a missing API key never blocks delivery.
func (a Aggregator) synthesiseDigestMeta(ctx context.Context, day time.Time, sections []news.SourceItems) (subject, summary string) {
	subject = "GoDaily - " + day.Format("January 2, 2006")
	if a.synthesiser == nil {
		return subject, ""
	}
	meta, err := a.synthesiser.Synthesise(ctx, day, sections)
	if err != nil {
		slog.WarnContext(ctx, "Synth digest meta failed, using static subject", "err", err)
		return subject, ""
	}
	return meta.Title, meta.Intro
}
