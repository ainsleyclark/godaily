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
	htmltemplate "html/template"
	"log/slog"
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/synth"
)

// Send generates a synth suggestion from sections (when available), appends it
// to the rendered bodies stored in issue, and ships the result via email. When
// a repository is configured and issue.ID > 0, the stored issue status is
// updated to reflect the outcome.
//
// sections may be nil; if so, or if no suggester is configured, the suggestion
// step is skipped and the stored HTML/text is sent as-is.
func (a Aggregator) Send(ctx context.Context, issue news.Issue, sections []news.SourceItems) error {
	if a.sendToAddress == "" {
		slog.WarnContext(ctx, "EMAIL_SEND_ADDRESS not set, skipping send")
		return nil
	}

	htmlBody := issue.HtmlBody
	textBody := issue.TextBody

	if len(sections) > 0 && a.suggester != nil {
		day := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
		s, err := a.suggester.Suggest(ctx, day, sections)
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

	if a.issues != nil && issue.ID > 0 {
		if _, err := a.issues.UpdateStatus(ctx, issue.ID, status, time.Now().UTC()); err != nil {
			slog.ErrorContext(ctx, "failed to update issue status", "err", err)
		}
	}

	return nil
}
