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
	"strings"
	"time"

	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleyclark/godaily/internal/synth"
)

// Send loads the draft digest for the given date, sends it to the
// configured address, and updates the stored issue status.
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

	if a.sendToAddress == "" {
		slog.WarnContext(ctx, "EMAIL_SEND_ADDRESS not set, skipping send")
		return nil
	}

	rendered := renderedDigest{
		Subject: issue.Subject,
		HTML:    issue.HtmlBody,
		Text:    issue.TextBody,
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

// SendSuggestion generates an AI post suggestion from the stored digest
// items for the given date and emails it to the owner address only.
func (a Aggregator) SendSuggestion(ctx context.Context, date time.Time) error {
	if a.suggester == nil {
		return errors.New("synth send requires ANTHROPIC_API_KEY")
	}
	if a.issues == nil || a.items == nil {
		return errors.New("synth send requires persistence (TURSO_URL not set)")
	}
	if a.sendToAddress == "" {
		slog.WarnContext(ctx, "EMAIL_SEND_ADDRESS not set, skipping synth send")
		return nil
	}

	slug := date.Format("2006-01-02")

	issue, err := a.issues.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("no digest found for %s — run `godaily collect` first", slug)
		}
		return fmt.Errorf("loading digest: %w", err)
	}

	sections, err := loadSections(ctx, a.items, issue.ID)
	if err != nil {
		return fmt.Errorf("loading items: %w", err)
	}

	if len(sections) == 0 {
		slog.InfoContext(ctx, "no items for synth suggestion, skipping")
		return nil
	}

	s, err := a.suggester.Suggest(ctx, date, sections)
	if err != nil {
		return fmt.Errorf("synth: %w", err)
	}

	return a.email.Send(ctx, email.SendEmailRequest{
		From:    "noreply@mail.ainsley.dev",
		To:      []string{a.sendToAddress},
		Subject: "GoDaily Synth - " + date.Format("2006-01-02"),
		Html:    suggestionHTML(s),
		Text:    s.Markdown(),
	})
}

// suggestionHTML renders a Suggestion as a minimal HTML email body.
func suggestionHTML(s synth.Suggestion) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<h3>Suggested post: %s</h3>\n", s.Date.Format("2006-01-02"))
	b.WriteString("<pre style=\"white-space: pre-wrap; font-family: inherit;\">")
	b.WriteString(htmltemplate.HTMLEscapeString(s.Post))
	b.WriteString("</pre>\n")
	if len(s.References) > 0 {
		b.WriteString("<h4>References</h4>\n<ul>\n")
		for _, r := range s.References {
			fmt.Fprintf(&b, "<li><a href=%q>%s</a> (%s)</li>\n",
				r.URL, htmltemplate.HTMLEscapeString(r.Title), r.Source)
		}
		b.WriteString("</ul>\n")
	}
	return b.String()
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
