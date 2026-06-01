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
	"github.com/ainsleyclark/godaily/pkg/services/social/internal/slackdata"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
)

// PublishDrafts walks every row with status='draft' and posts each to
// its platform, transitioning the row to published (or error). It is the
// single publish path for both featured and rotation kinds — the build
// cron generates every draft at 02:00, this runs at 11:00 to actually
// send them.
//
// Cancelled rows are filtered server-side by the status filter and so
// are never picked up.
//
// opts.Platforms restricts which platforms to publish. opts.DryRun is
// honored as a defensive no-op so CLI dry-runs do not accidentally
// promote drafts.
func (s *Service) PublishDrafts(ctx context.Context, opts social.PostOptions) ([]social.PostResult, error) {
	if !s.hasPosters() {
		slog.InfoContext(ctx, "Skipping publish — no posters configured")
		return nil, nil
	}

	draftStatus := social.PostStatusDraft
	drafts, err := s.posts.List(ctx, social.PostListOptions{Status: &draftStatus})
	if err != nil {
		return nil, errors.Wrap(err, "loading drafts")
	}
	if kindsFilter := kindFilter(opts.Kinds); kindsFilter != nil {
		drafts = filterByKinds(drafts, kindsFilter)
	}
	if len(drafts) == 0 {
		slog.InfoContext(ctx, "No drafts to publish")
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
				fmt.Sprintf("Publishing %s %s draft failed", slackdata.PlatformLabel(p), draft.Kind), err,
			))
			errStatus := social.PostStatusError
			if _, uerr := s.posts.Update(ctx, draft.ID, social.PostUpdate{Status: &errStatus}); uerr != nil {
				slog.WarnContext(ctx, "Failed to mark draft as errored", "id", draft.ID, "err", uerr)
			}
			results = append(results, res)
			continue
		}
		res.PostURL = result.PostURL

		now := time.Now().UTC()
		publishedStatus := social.PostStatusPublished
		if _, err = s.posts.Update(ctx, draft.ID, social.PostUpdate{
			Status:      &publishedStatus,
			PublishedAt: &now,
			PostURL:     &result.PostURL,
		}); err != nil {
			res.Err = errors.Wrap(err, "marking draft published")
			errs = append(errs, fmt.Errorf("%s: %w", p, res.Err))
			s.notifyFailure(ctx, slack.Error(
				fmt.Sprintf("Recording %s publish failed", slackdata.PlatformLabel(p)), err,
			))
			results = append(results, res)
			continue
		}

		slog.InfoContext(ctx, "Social draft published",
			"platform", p, "kind", string(draft.Kind), "url", result.PostURL)
		results = append(results, res)
	}

	if !opts.DryRun {
		s.notifyPublishSummary(ctx, opts.Date.UTC(), results)
	}

	if len(errs) > 0 {
		return results, stderrors.Join(errs...)
	}
	return results, nil
}

// notifyPublishSummary pings Slack once per PublishDrafts run with the
// "drafts published" card.
func (s *Service) notifyPublishSummary(ctx context.Context, date time.Time, results []social.PostResult) {
	if s.slack == nil {
		return
	}
	if req, ok := slackdata.DraftsPublished(date, results); ok {
		s.slack.MustSend(ctx, req)
	}
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

// kindFilter returns a set keyed by PostKind for fast membership checks,
// or nil when no kind restriction applies (publish every kind).
func kindFilter(wanted []social.PostKind) map[social.PostKind]bool {
	if len(wanted) == 0 {
		return nil
	}
	out := make(map[social.PostKind]bool, len(wanted))
	for _, k := range wanted {
		out[k] = true
	}
	return out
}

// filterByKinds returns the subset of rows whose Kind is in wanted.
// In-memory filtering is fine: today's draft set is small (one row per
// configured platform per drafted kind — order-of-magnitude single
// digits) and the alternative is paying the cost of a sqlc IN-clause.
func filterByKinds(rows []social.Post, wanted map[social.PostKind]bool) []social.Post {
	out := make([]social.Post, 0, len(rows))
	for _, r := range rows {
		if wanted[r.Kind] {
			out = append(out, r)
		}
	}
	return out
}
