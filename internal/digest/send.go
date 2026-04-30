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
	"time"

	"github.com/ainsleyclark/godaily/internal/news"
)

// Send ships the rendered digest in issue via email and, when a repository is
// configured and issue.ID > 0, updates the stored issue status to reflect the
// outcome.
func (a Aggregator) Send(ctx context.Context, issue news.Issue) error {
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

	if a.issues != nil && issue.ID > 0 {
		if _, err := a.issues.UpdateStatus(ctx, issue.ID, status, time.Now().UTC()); err != nil {
			slog.ErrorContext(ctx, "failed to update issue status", "err", err)
		}
	}

	return nil
}
