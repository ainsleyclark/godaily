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
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	"sort"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleyclark/godaily/internal/synth"
)

// Send loads the draft digest for the given date, generates a synth
// suggestion when items are present, sends the result via email, and
// updates the stored issue status to reflect the outcome.
func (a Aggregator) Send(ctx context.Context, date time.Time) error {
	if a.issues == nil || a.items == nil {
		return errors.New("send requires persistence (TURSO_URL not set)")
	}

	slug := date.Format("2006-01-02")

	issue, err := a.issues.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("no digest found for %s — run `godaily collect` first", slug)
		}
		return fmt.Errorf("loading digest: %w", err)
	}
	if issue.Status != news.IssueStatusDraft {
		return fmt.Errorf("digest for %s has status %q, expected %q", slug, issue.Status, news.IssueStatusDraft)
	}

	sections, err := loadSections(ctx, a.items, issue.ID)
	if err != nil {
		return fmt.Errorf("loading items: %w", err)
	}

	if a.sendToAddress == "" {
		slog.WarnContext(ctx, "EMAIL_SEND_ADDRESS not set, skipping send")
		return nil
	}

	htmlBody := issue.HtmlBody
	textBody := issue.TextBody

	if len(sections) > 0 && a.suggester != nil {
		s, err := a.suggester.Suggest(ctx, date, sections)
		switch {
		case errors.Is(err, synth.ErrNoItems):
			slog.InfoContext(ctx, "synth skipped: no items to summarise")
		case err != nil:
			slog.ErrorContext(ctx, "synth failed", "err", err)
		default:
			htmlBody += "\n<hr>\n<h3>Suggested post</h3>\n<pre style=\"white-space: pre-wrap; font-family: inherit;\">" +
				htmltemplate.HTMLEscapeString(s.Post) + "</pre>\n"
			textBody += "\nSuggested post\n==============\n" + s.Post + "\n"
		}
	}

	rendered := renderedDigest{
		Subject: issue.Subject,
		HTML:    htmlBody,
		Text:    textBody,
	}

	status := news.IssueStatusSent
	if err := a.sendDigest(ctx, rendered); err != nil {
		slog.ErrorContext(ctx, "failed to send digest email", "err", err)
		status = news.IssueStatusError
	}

	if _, err := a.issues.UpdateStatus(ctx, issue.ID, status, time.Now().UTC()); err != nil {
		slog.ErrorContext(ctx, "failed to update issue status", "err", err)
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
