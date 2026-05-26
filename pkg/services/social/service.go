// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package social orchestrates social media posting. Two entry points share
// the per-platform publish loop:
//
//   - Post: the daily "featured" path. Loads the digest for a date, picks
//     the featured item via AI, reframes for each enabled platform.
//   - Rotate: the Tue/Fri "rotation" path. Walks a kind-specific candidate
//     list (self-release, spotlight, cta, recap) and publishes the first
//     eligible one.
package social

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
	"github.com/ainsleyclark/godaily/pkg/store"
)

// Service publishes social media posts for both the daily featured slot
// and the Tue/Fri rotation slot.
type Service struct {
	posters    []platform.Poster
	prompter   ai.Prompter
	issues     digest.IssueRepository
	items      news.ItemRepository
	posts      social.PostRepository
	slack      slack.Sender
	reframers  map[social.Platform]reframer
	candidates []Candidate
}

// reframer reframes a featured item for one platform. Function-typed so
// tests can inject deterministic text without going through the AI.
type reframer func(ctx context.Context, p ai.Prompter, f featured.Featured) (string, error)

// defaultReframers maps each Platform to its production reframing prompt.
func defaultReframers() map[social.Platform]reframer {
	return map[social.Platform]reframer{
		social.Bluesky:  featured.Bluesky,
		social.LinkedIn: featured.LinkedIn,
		social.Mastodon: featured.Mastodon,
	}
}

// New creates a new social Service. posters may be empty (nothing to post);
// the service errors if prompter, issues, items, or posts are nil.
// slackSender may be nil to disable Slack notifications. Rotation candidates
// must be wired separately via WithCandidates if Rotate will be called.
func New(
	posters []platform.Poster,
	prompter ai.Prompter,
	issues digest.IssueRepository,
	items news.ItemRepository,
	posts social.PostRepository,
	slackSender slack.Sender,
) (*Service, error) {
	if prompter == nil {
		return nil, errors.New("social: ai.Prompter is required")
	}
	if issues == nil || items == nil {
		return nil, errors.New("social: issue and item repositories are required")
	}
	if posts == nil {
		return nil, errors.New("social: social post repository is required")
	}
	return &Service{
		posters:   posters,
		prompter:  prompter,
		issues:    issues,
		items:     items,
		posts:     posts,
		slack:     slackSender,
		reframers: defaultReframers(),
	}, nil
}

// WithCandidates registers the rotation candidates the service offers when
// Rotate is called. Order matters per-day but final selection is by the
// day-aware logic in rotation.go.
func (s *Service) WithCandidates(cs ...Candidate) *Service {
	s.candidates = cs
	return s
}

// HasPosters reports whether the service has any platforms configured.
// Useful for callers that want to short-circuit when no creds are set.
func (s *Service) HasPosters() bool {
	return len(s.posters) > 0
}

// Post is the entry point for the daily featured slot. It:
//  1. Loads the issue for Date (returns store.ErrNotFound if there is none).
//  2. Loads the issue's items.
//  3. Picks the day's featured item via the AI prompter.
//  4. For each enabled poster: skips if already posted today; otherwise
//     reframes the featured item for that platform, publishes it, and
//     records the result.
func (s *Service) Post(ctx context.Context, opts social.PostOptions) ([]social.PostResult, error) {
	if len(s.posters) == 0 {
		slog.InfoContext(ctx, "Skipping social — no posters configured")
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
		slog.InfoContext(ctx, "Skipping social — no items for issue", "issue", issue.ID)
		return nil, nil
	}

	feat, err := featured.Feature(ctx, s.prompter, date, rows)
	if err != nil {
		s.notifyFailure(ctx, "AI feature pick failed: "+err.Error())
		return nil, errors.Wrap(err, "feature")
	}

	slog.InfoContext(
		ctx, "Selected featured item",
		"title", feat.Title, "url", feat.URL, "tag", string(feat.Tag),
	)

	wanted := selectPosters(s.posters, opts.Platforms)
	issueID := issue.ID

	return s.publish(ctx, publishCtx{
		platforms: wanted,
		dryRun:    opts.DryRun,
		kind:      social.PostKindFeatured,
		issueID:   &issueID,
		generate: func(_ context.Context, p social.Platform) (string, error) {
			reframe, ok := s.reframers[p]
			if !ok {
				return "", fmt.Errorf("no reframer registered for platform %s", p)
			}
			return reframe(ctx, s.prompter, feat)
		},
		// Featured posts use (issue_id, platform) for idempotency.
		skipIfPosted: func(ctx context.Context, p string) (bool, error) {
			return s.posts.HasPosted(ctx, issueID, p)
		},
	})
}

// publishCtx bundles the inputs of one publish() loop. Each rotation
// candidate produces one of these on its way through Rotate; the featured
// path constructs one inline.
type publishCtx struct {
	platforms []platform.Poster
	dryRun    bool
	kind      social.PostKind
	issueID   *int64
	subject   string

	// generate returns the post text for a given platform. Candidates may
	// ignore the platform and return identical text everywhere; the
	// featured path uses the platform reframers.
	generate func(ctx context.Context, platform social.Platform) (string, error)

	// skipIfPosted is the per-row idempotency check. Returning true skips
	// the platform without an error.
	skipIfPosted func(ctx context.Context, platform string) (bool, error)
}

// publish runs the per-platform reframe → post → persist loop. It is the
// shared core of both the featured (Post) and rotation (Rotate) paths.
//
// Per-platform errors are accumulated, not fatal. The slack notifier is
// pinged on any failure.
func (s *Service) publish(ctx context.Context, pc publishCtx) ([]social.PostResult, error) {
	results := make([]social.PostResult, 0, len(pc.platforms))
	var errs []error

	for _, poster := range pc.platforms {
		res := s.publishOne(ctx, poster, pc)
		results = append(results, res)
		if res.Err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", res.Platform, res.Err))
			s.notifyFailure(ctx, fmt.Sprintf("Social %s post to %s failed: %s", pc.kind, res.Platform, res.Err))
		}
	}

	if !pc.dryRun {
		s.notifySuccess(ctx, pc, results)
	}

	if len(errs) > 0 {
		return results, stderrors.Join(errs...)
	}
	return results, nil
}

func (s *Service) publishOne(ctx context.Context, poster platform.Poster, pc publishCtx) social.PostResult {
	p := poster.Platform()
	res := social.PostResult{Platform: p, Kind: pc.kind}

	if !pc.dryRun && pc.skipIfPosted != nil {
		posted, err := pc.skipIfPosted(ctx, p.String())
		if err != nil {
			res.Err = errors.Wrap(err, "checking idempotency")
			return res
		}
		if posted {
			res.Skipped = true
			slog.InfoContext(
				ctx, "Skipping platform — already posted",
				"platform", p, "kind", string(pc.kind),
			)
			return res
		}
	}

	text, err := pc.generate(ctx, p)
	if err != nil {
		res.Err = errors.Wrap(err, "generate")
		return res
	}
	res.Text = text

	if pc.dryRun {
		slog.InfoContext(
			ctx, "Dry-run: skipping post + DB write",
			"platform", p, "kind", string(pc.kind), "chars", len(text),
		)
		return res
	}

	result, err := poster.Post(ctx, text)
	if err != nil {
		res.Err = errors.Wrap(err, "poster.Post")
		return res
	}
	res.PostURL = result.PostURL

	if _, err := s.posts.Create(ctx, social.Post{
		IssueID:  pc.issueID,
		Kind:     pc.kind,
		Subject:  pc.subject,
		Platform: p.String(),
		Text:     text,
		PostURL:  result.PostURL,
	}); err != nil {
		// The platform post already succeeded — failing the DB write is
		// bad but not fatal (we'll notify Slack so we can backfill).
		res.Err = errors.Wrap(err, "recording social_post")
		return res
	}

	slog.InfoContext(
		ctx, "Social post published",
		"platform", p, "kind", string(pc.kind), "url", result.PostURL,
	)
	return res
}

// selectPosters narrows the configured posters to those requested in opts.
// When wanted is empty the full slice is returned unchanged.
func selectPosters(all []platform.Poster, wanted []social.Platform) []platform.Poster {
	if len(wanted) == 0 {
		return all
	}

	wantedSet := make(map[social.Platform]bool, len(wanted))
	for _, p := range wanted {
		wantedSet[p] = true
	}

	out := make([]platform.Poster, 0, len(all))
	for _, p := range all {
		if wantedSet[p.Platform()] {
			out = append(out, p)
		}
	}
	return out
}

func (s *Service) notifyFailure(ctx context.Context, msg string) {
	if s.slack != nil {
		s.slack.MustSend(ctx, msg)
	}
}

// notifySuccess pings Slack once per publish run with a clickable link to
// every platform that posted successfully. Skipped (idempotent) and
// failed platforms are omitted; failures are already covered by
// notifyFailure inside the loop. A no-op when no platform succeeded or
// when the slack sender is not configured.
func (s *Service) notifySuccess(ctx context.Context, pc publishCtx, results []social.PostResult) {
	if s.slack == nil {
		return
	}

	lines := make([]string, 0, len(results))
	for _, r := range results {
		if r.Err != nil || r.Skipped || r.PostURL == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("• %s: <%s|view post>", platformLabel(r.Platform), r.PostURL))
	}
	if len(lines) == 0 {
		return
	}

	header := fmt.Sprintf("Social post published — %s", pc.kind)
	if pc.subject != "" {
		header += fmt.Sprintf(" (%s)", pc.subject)
	}

	s.slack.MustSend(ctx, header+"\n"+strings.Join(lines, "\n"))
}

// platformLabel returns the human-friendly name for a platform used in
// Slack notifications.
func platformLabel(p social.Platform) string {
	switch p {
	case social.Bluesky:
		return "Bluesky"
	case social.LinkedIn:
		return "LinkedIn"
	case social.Mastodon:
		return "Mastodon"
	default:
		return string(p)
	}
}
