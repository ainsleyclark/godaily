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
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
)

// SendPreview loads the draft digest for the given date, sends it to the
// admin address along with an AI synth suggestion, and leaves the issue in
// draft status so SendDigest can still dispatch it to subscribers later.
func (a Aggregator) SendPreview(ctx context.Context, date time.Time) error {
	slug := date.Format("2006-01-02")

	slog.InfoContext(ctx, "Preparing to send preview digest", "slug", slug)

	issue, sections, err := a.loadDraftDigest(ctx, slug, false)
	if err != nil {
		return err
	}

	canonicalURL := env.AppURL + "/issues/" + issue.Slug + "/"

	rendered, err := renderDigest(digestOptions{
		Day:          date,
		Subject:      issue.Subject,
		Intro:        issue.Summary,
		Sources:      sections,
		CanonicalURL: canonicalURL,
	})
	if err != nil {
		return errors.Wrap(err, "rendering digest")
	}

	issueTag := email.Tag{Name: email.TagIssueID, Value: strconv.FormatInt(issue.ID, 10)}
	if err = a.sendRendered(ctx, a.adminEmailAddress, rendered, []email.Tag{issueTag}); err != nil {
		return errors.Wrap(err, "sending preview digest")
	}

	// Synth suggestion is best-effort: a missing prompter or AI error should
	// not block the owner from receiving the digest preview.
	if suggErr := a.SendSuggestion(ctx, date); suggErr != nil {
		slog.WarnContext(ctx, "Failed to send synth suggestion during preview", "err", suggErr)
	}

	return nil
}
