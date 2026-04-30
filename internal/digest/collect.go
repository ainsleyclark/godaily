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
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/synth"
)

// CollectOptions configures a Collect call.
type CollectOptions struct {
	// DryRun skips rendering and persisting the digest; only the raw
	// source items are returned.
	DryRun bool

	// Sources restricts the run to the given sources. If empty,
	// all registered sources (news.Sources) are used.
	Sources []news.Source

	// IncludeSynth, when true, calls the synth package after scoring
	// to draft suggested social posts and includes them in the digest.
	// A synth failure is logged but does not abort the digest.
	IncludeSynth bool
}

// Collect fetches Go news items published yesterday from all registered
// sources, scores and sorts them, optionally synthesises social copy, renders
// the digest and (unless DryRun) persists it as a draft issue in the database.
//
// Returns the persisted Issue (ID=0 when DryRun or no repository is set) and
// the raw SourceItems so callers can inspect or serialise them.
func (a Aggregator) Collect(ctx context.Context, opts CollectOptions) (news.Issue, []news.SourceItems, error) {
	day := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour) // Yesterday
	next := day.AddDate(0, 0, 1)

	sources := opts.Sources
	if len(sources) == 0 {
		sources = news.Sources
	}

	var results []news.SourceItems
	for _, src := range sources {
		fetched, err := a.fetchSource(ctx, src)
		if err != nil {
			slog.ErrorContext(ctx, "failed to fetch source", "source", src, "err", err)
			continue
		}
		si := news.SourceItems{Source: src}

		for _, item := range fetched {
			if item.Published.IsZero() {
				slog.ErrorContext(ctx, "item has zero published date", "source", src, "title", item.Title)
				continue
			}
			if item.Published.After(day) && item.Published.Before(next) {
				si.Items = append(si.Items, item)
			}
		}

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

	var suggestion *synth.Suggestion
	if opts.IncludeSynth && a.suggester != nil && !opts.DryRun {
		s, err := a.suggester.Suggest(ctx, day, results)
		switch {
		case errors.Is(err, synth.ErrNoItems):
			slog.InfoContext(ctx, "synth skipped: no items to summarise")
		case err != nil:
			slog.ErrorContext(ctx, "synth failed", "err", err)
		default:
			suggestion = &s
		}
	}

	if opts.DryRun || len(results) == 0 {
		return news.Issue{}, results, nil
	}

	rendered, err := renderDigest(day, results, suggestion)
	if err != nil {
		slog.ErrorContext(ctx, "failed to render digest", "err", err)
		return news.Issue{}, results, nil
	}

	issue := news.Issue{
		Slug:     day.Format("2006-01-02"),
		Subject:  rendered.Subject,
		HtmlBody: rendered.HTML,
		TextBody: rendered.Text,
		Status:   news.IssueStatusDraft,
		SentAt:   time.Now().UTC(),
	}

	if a.issues != nil {
		persisted, err := a.persistIssue(ctx, issue, results)
		if err != nil {
			slog.ErrorContext(ctx, "failed to persist issue", "err", err)
			return issue, results, nil
		}
		issue = persisted
	}

	return issue, results, nil
}
