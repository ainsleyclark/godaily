// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
// admin address, and leaves the issue in draft status so SendDigest can
// still dispatch it to subscribers later.
func (s Service) SendPreview(ctx context.Context, date time.Time) error {
	slug := date.Format("2006-01-02")

	slog.InfoContext(ctx, "Preparing to send preview digest", "slug", slug)

	issue, sections, err := s.loadDraftDigest(ctx, slug, false)
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
	if err = s.sendRendered(ctx, s.adminEmailAddress, rendered, []email.Tag{issueTag}); err != nil {
		return errors.Wrap(err, "sending preview digest")
	}

	return nil
}
