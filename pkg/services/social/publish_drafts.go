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
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// PublishDrafts runs the platform half of the featured pipeline: it loads
// today's draft featured posts and publishes each to its platform. On
// success the row transitions to status='published' with post_url and
// published_at populated; on failure it transitions to status='error' so
// a later retry can flip it back to draft.
//
// opts.Platforms restricts which platforms to publish. opts.DryRun is
// honored as a defensive no-op so CLI dry-runs do not accidentally
// promote drafts.
func (s *Service) PublishDrafts(ctx context.Context, opts social.PostOptions) ([]social.PostResult, error) {
	if !s.hasPosters() {
		slog.InfoContext(ctx, "Skipping publish — no posters configured")
		return nil, nil
	}

	date := opts.Date.UTC()
	slug := date.Format("2006-01-02")

	issue, err := s.issues.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			slog.InfoContext(ctx, "No digest found — skipping publish", "slug", slug)
			return nil, nil
		}
		return nil, errors.Wrap(err, "loading digest")
	}

	draftStatus := social.PostStatusDraft
	drafts, err := s.posts.List(ctx, social.PostListOptions{
		IssueID: &issue.ID,
		Status:  &draftStatus,
	})
	if err != nil {
		return nil, errors.Wrap(err, "loading drafts")
	}
	if len(drafts) == 0 {
		slog.InfoContext(ctx, "No drafts to publish", "issue", issue.ID)
		return nil, nil
	}

	postersByPlatform := postersByPlatformMap(s.posters)
	wantedFilter := platformFilter(opts.Platforms)

	results := make([]social.PostResult, 0, len(drafts))
	var errs []error

	for _, draft := range drafts {
		p := social.Platform(draft.Platform)
		if wantedFilter != nil && !wantedFilter[p] {
			continue
		}

		res := social.PostResult{Platform: p, Kind: draft.Kind, Text: draft.Text}

		poster, ok := postersByPlatform[p]
		if !ok {
			slog.InfoContext(ctx, "Skipping draft — no poster wired for platform", "platform", p)
			continue
		}

		if opts.DryRun {
			slog.InfoContext(ctx, "Dry-run: skipping publish", "platform", p, "chars", len(draft.Text))
			results = append(results, res)
			continue
		}

		var mentions []social.Mention
		if draft.MentionSource != "" {
			if profile, ok := social.ProfileFor(news.Source(draft.MentionSource)); ok {
				mentions = profile.Mentions
			}
		}

		result, err := poster.Post(ctx, platform.PostRequest{Text: draft.Text, Mentions: mentions})
		if err != nil {
			res.Err = errors.Wrap(err, "poster.Post")
			errs = append(errs, fmt.Errorf("%s: %w", p, res.Err))
			s.notifyFailure(ctx, slack.Error(
				fmt.Sprintf("Publishing %s draft failed", platformLabel(p)), err,
			))
			if _, uerr := s.posts.UpdateStatus(ctx, draft.ID, social.PostStatusError, nil, ""); uerr != nil {
				slog.WarnContext(ctx, "Failed to mark draft as errored", "id", draft.ID, "err", uerr)
			}
			results = append(results, res)
			continue
		}
		res.PostURL = result.PostURL

		now := time.Now().UTC()
		if _, err = s.posts.UpdateStatus(ctx, draft.ID, social.PostStatusPublished, &now, result.PostURL); err != nil {
			res.Err = errors.Wrap(err, "marking draft published")
			errs = append(errs, fmt.Errorf("%s: %w", p, res.Err))
			s.notifyFailure(ctx, slack.Error(
				fmt.Sprintf("Recording %s publish failed", platformLabel(p)), err,
			))
			results = append(results, res)
			continue
		}

		slog.InfoContext(ctx, "Social draft published", "platform", p, "url", result.PostURL)
		results = append(results, res)
	}

	if !opts.DryRun {
		s.notifySuccess(ctx, publishCtx{kind: social.PostKindFeatured, subject: "Featured for " + slug}, results)
	}

	if len(errs) > 0 {
		return results, stderrors.Join(errs...)
	}
	return results, nil
}

// postersByPlatformMap inverts the posters slice for O(1) lookup at
// publish time when matching a draft row's platform to its Poster.
func postersByPlatformMap(posters []platform.Poster) map[social.Platform]platform.Poster {
	out := make(map[social.Platform]platform.Poster, len(posters))
	for _, p := range posters {
		out[p.Platform()] = p
	}
	return out
}

// platformFilter turns a wanted slice into a set, or nil when no
// restriction applies.
func platformFilter(wanted []social.Platform) map[social.Platform]bool {
	if len(wanted) == 0 {
		return nil
	}
	out := make(map[social.Platform]bool, len(wanted))
	for _, p := range wanted {
		out[p] = true
	}
	return out
}
