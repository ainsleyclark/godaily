// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// DraftAll runs the AI half of every social pipeline at digest Build
// time. It generates one featured draft per configured platform for
// today's issue and — on rotation days — generates the rotation draft
// for that weekday. No platform HTTP happens here.
//
// Re-running for the same date is safe: existing featured drafts for
// the issue are cleared first, and stale rotation drafts of the
// candidate's kind are wiped before regeneration. Subjects already
// published OR cancelled are skipped via HasPostedOrCancelledBySubject
// so a deliberately-cancelled rotation does not come back.
//
// On opts.DryRun, the AI work runs end-to-end but no draft rows are
// persisted — useful for CLI smoke tests.
func (s *Service) DraftAll(ctx context.Context, opts social.PostOptions) ([]social.PostResult, error) {
	if !s.hasPosters() {
		slog.InfoContext(ctx, "Skipping draft — no posters configured")
		return nil, nil
	}

	var all []social.PostResult
	var errs []error

	featuredResults, ferr := s.draftFeatured(ctx, opts)
	all = append(all, featuredResults...)
	if ferr != nil {
		errs = append(errs, ferr)
	}

	rotationResults, rerr := s.draftRotation(ctx, opts)
	all = append(all, rotationResults...)
	if rerr != nil {
		errs = append(errs, rerr)
	}

	if len(errs) > 0 {
		return all, stderrors.Join(errs...)
	}
	return all, nil
}

// draftFeatured generates draft featured rows for opts.Date. Internal to
// the Service — callers go through DraftAll.
func (s *Service) draftFeatured(ctx context.Context, opts social.PostOptions) ([]social.PostResult, error) {
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
		slog.InfoContext(ctx, "Skipping featured draft — no items for issue", "issue", issue.ID)
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

		slog.InfoContext(ctx, "Featured draft persisted", "platform", p, "chars", len(text))
		results = append(results, res)
	}

	if len(errs) > 0 {
		return results, stderrors.Join(errs...)
	}
	return results, nil
}

// draftRotation walks the day's rotation candidates, picks the first
// eligible one, and persists a draft row per configured platform. Mirrors
// Rotate but with draftOnly=true so no platform HTTP happens. The
// publish cron picks the rows up at 11:00.
func (s *Service) draftRotation(ctx context.Context, opts social.PostOptions) ([]social.PostResult, error) {
	if len(s.candidates) == 0 {
		return nil, nil
	}

	now := opts.Date.UTC()
	candidates, err := s.pickCandidates(social.RotateOptions{Now: now})
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	for _, cand := range candidates {
		cctx, ok, err := cand.Eligible(ctx, now)
		if err != nil {
			s.notifyFailure(ctx, slack.Error(
				"Rotation eligibility check failed — "+string(cand.Kind()), err,
			))
			return nil, errors.Wrapf(err, "eligibility for %s", cand.Kind())
		}
		if !ok {
			slog.InfoContext(ctx, "Rotation candidate not eligible", "kind", string(cand.Kind()))
			continue
		}

		slog.InfoContext(
			ctx, "Rotation candidate eligible — drafting",
			"kind", string(cand.Kind()), "subject", cctx.Subject,
		)

		if !opts.DryRun {
			if derr := s.posts.DeleteDraftsByKind(ctx, cand.Kind()); derr != nil {
				return nil, errors.Wrapf(derr, "clearing draft rows for %s", cand.Kind())
			}
		}

		wanted := selectPosters(s.posters, opts.Platforms)
		return s.publish(ctx, publishCtx{
			platforms: wanted,
			dryRun:    opts.DryRun,
			draftOnly: true,
			kind:      cand.Kind(),
			subject:   cctx.Subject,
			generate: func(ctx context.Context, p social.Platform) (string, error) {
				text, err := cand.Generate(ctx, s.prompter, p, cctx)
				if err != nil {
					return "", err
				}
				if cctx.Kind == social.PostKindNewSource || cctx.Kind == social.PostKindRecap {
					text = appendSubscribeLine(text, p, string(cctx.Kind))
				}
				return text, nil
			},
			skipIfPosted: subjectIdempotency(s.posts, cctx.Subject),
			mentions:     cctx.Mentions,
		})
	}

	slog.InfoContext(ctx, "Rotation: no eligible candidate to draft", "weekday", now.Weekday())
	return nil, nil
}
