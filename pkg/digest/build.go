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

	"github.com/ainsleyclark/godaily/pkg/digest/prompts"
	"github.com/ainsleyclark/godaily/pkg/news"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Build loads collected items for the appropriate date window, ranks and
// deduplicates them, runs AI synthesis, and persists a draft Issue with the
// items associated. If a draft already exists for the date's slug it is deleted
// and rebuilt. On any failure a Slack notification is sent.
func (a Aggregator) Build(ctx context.Context, date time.Time) error {
	today := date.UTC().Truncate(24 * time.Hour)
	start, end := buildWindow(today)
	slug := today.Format("2006-01-02")

	slog.InfoContext(ctx, "Building digest", "slug", slug, "start", start.Format("2006-01-02"), "end", end.Format("2006-01-02"))

	items, err := a.items.List(ctx, news.ItemListOptions{From: &start, To: &end})
	if err != nil {
		return a.buildErr(ctx, errors.Wrap(err, "listing items"))
	}

	if len(items) == 0 {
		slog.WarnContext(ctx, "No items found for build window, skipping", "slug", slug)
		return nil
	}

	sections := groupIntoSections(items)

	subject, summary := a.synthesiseDigestMeta(ctx, today, sections)

	existing, lookupErr := a.issues.FindBySlug(ctx, slug)
	switch {
	case lookupErr == nil:
		slog.InfoContext(ctx, "Replacing existing draft", "slug", slug)
		if err = a.items.DeleteByIssue(ctx, existing.ID); err != nil {
			return a.buildErr(ctx, errors.Wrap(err, "deleting existing items"))
		}
		if _, err = a.issues.Delete(ctx, existing.ID); err != nil {
			return a.buildErr(ctx, errors.Wrap(err, "deleting existing issue"))
		}
	case !errors.Is(lookupErr, store.ErrNotFound):
		return a.buildErr(ctx, errors.Wrap(lookupErr, "checking existing issue"))
	}

	issue := news.Issue{
		Slug:    slug,
		Subject: subject,
		Summary: summary,
		Status:  news.IssueStatusDraft,
		SentAt:  today,
	}

	created, err := a.issues.Create(ctx, issue)
	if err != nil {
		return a.buildErr(ctx, errors.Wrap(err, "creating issue"))
	}

	var position int
	for _, section := range sections {
		for _, item := range section.Items {
			position++
			item.Source = section.Source
			id := created.ID
			if _, err = a.items.Create(ctx, &id, position, item); err != nil {
				return a.buildErr(ctx, errors.Wrap(err, "associating item"))
			}
		}
	}

	slog.InfoContext(ctx, "Built draft issue", "slug", slug, "items", position)

	return nil
}

// buildWindow returns the date range of items to include in the digest.
// On Monday, it covers the previous Friday through Monday (4-day window).
// Tuesday through Friday, it covers only the previous day.
// today must already be truncated to midnight UTC.
func buildWindow(today time.Time) (start, end time.Time) {
	if today.Weekday() == time.Monday {
		return today.AddDate(0, 0, -3), today
	}
	return today.AddDate(0, 0, -1), today
}

// groupIntoSections groups a flat item list into SourceItems slices,
// preserving per-source score ordering and deduplicating by URL.
func groupIntoSections(items []news.Item) []news.SourceItems {
	seen := make(map[string]struct{})
	order := make([]news.Source, 0)
	bySource := make(map[news.Source]*news.SourceItems)

	for _, item := range items {
		if _, dup := seen[item.URL]; dup {
			continue
		}
		seen[item.URL] = struct{}{}

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

	return sections
}

func (a Aggregator) buildErr(ctx context.Context, err error) error {
	if a.slack != nil {
		a.slack.MustSend(ctx, "Build failed: "+err.Error())
	}
	return err
}

// synthesiseDigestMeta calls the prompter to generate the email subject title
// and intro paragraph. On failure it logs a warning and returns static fallbacks
// so a missing API key never blocks delivery.
func (a Aggregator) synthesiseDigestMeta(ctx context.Context, day time.Time, sections []news.SourceItems) (subject, summary string) {
	subject = "GoDaily - " + day.Format("January 2, 2006")
	if a.prompter == nil {
		return subject, ""
	}
	meta, err := prompts.Synthesise(ctx, a.prompter, day, sections)
	if err != nil {
		slog.WarnContext(ctx, "Synth digest meta failed, using static subject", "err", err)
		if a.slack != nil {
			a.slack.MustSend(ctx, "AI synthesis failed: "+err.Error())
		}
		return subject, ""
	}
	return meta.Title, meta.Intro
}
