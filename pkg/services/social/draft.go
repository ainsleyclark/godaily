// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// DraftFeatured runs the AI half of the featured pipeline: it loads the
// issue for opts.Date, picks the day's featured item, reframes it for
// each configured platform, and writes one draft row per platform to
// social_posts. No platform HTTP happens here. Existing drafts for the
// issue are cleared first so a re-run of Build for the same date cleanly
// replaces the previous attempt.
//
// On opts.DryRun, the AI work runs end-to-end but no draft rows are
// persisted — useful for CLI smoke tests.
func (s *Service) DraftFeatured(ctx context.Context, opts social.PostOptions) ([]social.PostResult, error) {
	if !s.hasPosters() {
		slog.InfoContext(ctx, "Skipping draft — no posters configured")
		return nil, nil
	}

	date := opts.Date.UTC()
	slug := date.Format("2006-01-02")

	issue, err := s.issues.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("no digest found for %s — run `godaily collect` first", slug)
		}
		return nil, errors.Wrap(err, "loading digest")
	}

	rows, err := s.items.List(ctx, news.ItemListOptions{IssueID: &issue.ID})
	if err != nil {
		return nil, errors.Wrap(err, "loading items")
	}
	if len(rows) == 0 {
		slog.InfoContext(ctx, "Skipping draft — no items for issue", "issue", issue.ID)
		return nil, nil
	}

	feat, err := featured.Feature(ctx, s.prompter, date, rows)
	if err != nil {
		s.notifyFailure(ctx, slack.Error("AI feature pick failed", err))
		return nil, errors.Wrap(err, "feature")
	}

	slog.InfoContext(
		ctx, "Selected featured item for drafts",
		"title", feat.Title, "url", feat.URL, "tag", string(feat.Tag),
	)

	if !opts.DryRun {
		if err = s.posts.DeleteDraftsByIssue(ctx, issue.ID); err != nil {
			return nil, errors.Wrap(err, "clearing existing drafts")
		}
	}

	wanted := selectPosters(s.posters, opts.Platforms)
	results := make([]social.PostResult, 0, len(wanted))
	var errs []error

	for _, poster := range wanted {
		p := poster.Platform()
		res := social.PostResult{Platform: p, Kind: social.PostKindFeatured}

		reframe, ok := s.reframers[p]
		if !ok {
			res.Err = fmt.Errorf("no reframer registered for platform %s", p)
			errs = append(errs, fmt.Errorf("%s: %w", p, res.Err))
			s.notifyFailure(ctx, slack.Error(
				fmt.Sprintf("Drafting %s failed", platformLabel(p)), res.Err,
			))
			results = append(results, res)
			continue
		}

		text, err := reframe(ctx, s.prompter, feat)
		if err != nil {
			res.Err = errors.Wrap(err, "reframer")
			errs = append(errs, fmt.Errorf("%s: %w", p, res.Err))
			s.notifyFailure(ctx, slack.Error(
				fmt.Sprintf("Drafting %s failed", platformLabel(p)), res.Err,
			))
			results = append(results, res)
			continue
		}
		res.Text = text

		if opts.DryRun {
			slog.InfoContext(ctx, "Dry-run: skipping draft persist", "platform", p, "chars", len(text))
			results = append(results, res)
			continue
		}

		issueID := issue.ID
		if _, err = s.posts.Create(ctx, social.Post{
			IssueID:       &issueID,
			Kind:          social.PostKindFeatured,
			Platform:      p.String(),
			Text:          text,
			Status:        social.PostStatusDraft,
			MentionSource: string(feat.Source),
		}); err != nil {
			res.Err = errors.Wrap(err, "persisting draft")
			errs = append(errs, fmt.Errorf("%s: %w", p, res.Err))
			s.notifyFailure(ctx, slack.Error(
				fmt.Sprintf("Persisting %s draft failed", platformLabel(p)), res.Err,
			))
			results = append(results, res)
			continue
		}

		slog.InfoContext(ctx, "Social draft persisted", "platform", p, "chars", len(text))
		results = append(results, res)
	}

	s.notifyDraftSuccess(ctx, date, results)

	if len(errs) > 0 {
		return results, stderrors.Join(errs...)
	}
	return results, nil
}

// notifyDraftSuccess pings Slack once per DraftFeatured run with the
// drafted text per platform, mirroring the digest preview Slack ping.
// Acts as a passive review window: the owner can spot anything off
// before the 11:00 publish cron fires.
func (s *Service) notifyDraftSuccess(ctx context.Context, date time.Time, results []social.PostResult) {
	if s.slack == nil {
		return
	}

	type preview struct {
		platform string
		text     string
	}
	var previews []preview
	for _, r := range results {
		if r.Err != nil || r.Text == "" {
			continue
		}
		previews = append(previews, preview{platform: platformLabel(r.Platform), text: r.Text})
	}
	if len(previews) == 0 {
		return
	}

	body := ""
	for _, p := range previews {
		body += fmt.Sprintf("*%s*\n%s\n\n", p.platform, p.text)
	}

	s.slack.MustSend(ctx, slack.Info(
		"Social drafts for "+date.Format("2006-01-02"),
		body,
	))
}
