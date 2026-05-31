// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/social/candidate"
)

// Rotate walks the day's candidate list (or just ForceKind), picks the
// first eligible one, and publishes it across the configured platforms.
//
//   - Tuesday: new_source → spotlight → cta → no-op.
//   - Wednesday: community (a Go conference or meetup), 2:1 meetup:conf
//     rotation through the curated lists in pkg/data.
//   - Friday: recap (only). No fallback — Friday is recap day; if there's
//     no click data, the slot stays quiet.
//   - Other days: no-op unless ForceKind is set.
func (s *Service) Rotate(ctx context.Context, opts social.RotateOptions) ([]social.PostResult, error) {
	if !s.hasPosters() {
		slog.InfoContext(ctx, "Skipping rotation — no posters configured")
		return nil, nil
	}
	if len(s.candidates) == 0 {
		slog.InfoContext(ctx, "Skipping rotation — no candidates registered")
		return nil, nil
	}

	candidates, err := s.pickCandidates(opts)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		slog.InfoContext(ctx, "Skipping rotation — wrong day", "weekday", opts.Now.UTC().Weekday())
		return nil, nil
	}

	now := opts.Now.UTC()
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
			ctx, "Rotation candidate eligible",
			"kind", string(cand.Kind()), "subject", cctx.Subject, "url", cctx.URL,
		)

		wanted := selectPosters(s.posters, opts.Platforms)
		return s.publish(ctx, publishCtx{
			platforms: wanted,
			dryRun:    opts.DryRun,
			kind:      cand.Kind(),
			subject:   cctx.Subject,
			generate: func(ctx context.Context, p social.Platform) (string, error) {
				return cand.Generate(ctx, s.prompter, p, cctx)
			},
			skipIfPosted: subjectIdempotency(s.posts, cctx.Subject),
			mentions:     cctx.Mentions,
		})
	}

	slog.InfoContext(ctx, "Rotation: no eligible candidate", "weekday", now.Weekday())

	return nil, nil
}

// pickCandidates returns the candidate list for the day, or honors
// ForceKind. Returns nil when the day is not a rotation day.
func (s *Service) pickCandidates(opts social.RotateOptions) ([]candidate.Candidate, error) {
	if opts.ForceKind != "" {
		c := candidateByKind(s.candidates, opts.ForceKind)
		if c == nil {
			return nil, errors.Errorf("no candidate registered for kind %q", opts.ForceKind)
		}
		return []candidate.Candidate{c}, nil
	}

	weekday := opts.Now.UTC().Weekday()
	switch weekday {
	case time.Tuesday:
		return orderedByKinds(
			s.candidates,
			social.PostKindNewSource,
			social.PostKindSpotlight,
			social.PostKindCTA,
		), nil
	case time.Wednesday:
		return orderedByKinds(s.candidates, social.PostKindCommunity), nil
	case time.Friday:
		return orderedByKinds(s.candidates, social.PostKindRecap), nil
	default:
		return nil, nil
	}
}

// orderedByKinds returns the subset of candidates matching the given
// kinds, in the requested order. Missing candidates are silently dropped
// — useful when a deployment hasn't wired every kind.
func orderedByKinds(all []candidate.Candidate, kinds ...social.PostKind) []candidate.Candidate {
	out := make([]candidate.Candidate, 0, len(kinds))
	for _, k := range kinds {
		if c := candidateByKind(all, k); c != nil {
			out = append(out, c)
		}
	}
	return out
}

// subjectIdempotency returns a skipIfPosted check keyed off the
// candidate's Subject. An empty subject disables the check (caller is
// trusting the candidate's own eligibility logic).
func subjectIdempotency(posts social.PostRepository, subject string) func(ctx context.Context, platform string) (bool, error) {
	if subject == "" {
		return nil
	}
	return func(ctx context.Context, platform string) (bool, error) {
		return posts.HasPostedBySubject(ctx, subject, platform)
	}
}
