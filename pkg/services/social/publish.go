// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package social

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/env"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/services/internal/slackkit"
	"github.com/ainsleyclark/godaily/pkg/services/social/platform"
	"github.com/pkg/errors"
)

// publishCtx bundles the inputs of one publish() loop. Each rotation
// candidate produces one of these on its way through Rotate or DraftAll;
// the featured path constructs one inline.
type publishCtx struct {
	platforms []platform.Poster
	dryRun    bool
	kind      social.PostKind
	issueID   *int64
	subject   string

	// draftOnly, when true, skips the platform HTTP call and persists
	// the generated text with status='draft' instead of 'published'. The
	// build cron uses this so a human can review every post before the
	// publish cron fires.
	draftOnly bool

	// mentionSource, when non-empty, is persisted on the draft row so
	// PublishDrafts can re-attach the platform mentions at publish time
	// without re-running the AI feature pick. Only meaningful when
	// draftOnly is true.
	mentionSource string

	// generate returns the post text for a given platform. Candidates may
	// ignore the platform and return identical text everywhere; the
	// featured path uses the platform reframers.
	generate func(ctx context.Context, platform social.Platform) (string, error)

	// skipIfPosted is the per-row idempotency check. Returning true skips
	// the platform without an error.
	skipIfPosted func(ctx context.Context, platform string) (bool, error)

	// mentions are the platform-tagged identities the post should
	// reference. Each Poster filters by m.Platform and renders the
	// matching subset natively (LinkedIn → inline annotations; Bluesky
	// / Mastodon → ignored, their @-handles are baked into text).
	mentions []social.Mention
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
			s.notifyFailure(ctx, slack.Error(
				fmt.Sprintf("Social %s → %s failed", pc.kind, platformLabel(res.Platform)),
				res.Err,
			))
		}
	}

	// notifySuccess emits the "post is live" Slack ping. Draft writes
	// don't go live, so the per-kind notification is suppressed; the
	// build cron emits one summary Slack message covering every drafted
	// kind at once.
	if !pc.dryRun && !pc.draftOnly {
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

	if pc.draftOnly {
		if _, err = s.posts.Create(ctx, social.Post{
			IssueID:       pc.issueID,
			Kind:          pc.kind,
			Subject:       pc.subject,
			Platform:      p.String(),
			Text:          text,
			Status:        social.PostStatusDraft,
			MentionSource: pc.mentionSource,
		}); err != nil {
			res.Err = errors.Wrap(err, "persisting draft")
			return res
		}
		slog.InfoContext(
			ctx, "Social draft persisted",
			"platform", p, "kind", string(pc.kind), "chars", len(text),
		)
		return res
	}

	result, err := poster.Post(ctx, platform.PostRequest{Text: text, Mentions: pc.mentions})
	if err != nil {
		res.Err = errors.Wrap(err, "poster.Post")
		return res
	}
	res.PostURL = result.PostURL

	if _, err = s.posts.Create(ctx, social.Post{
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

func (s *Service) notifyFailure(ctx context.Context, req slack.Request) {
	if s.slack != nil {
		s.slack.MustSend(ctx, req)
	}
}

// notifySuccess pings Slack once per publish run with a rich card showing,
// per platform that posted successfully, the live post copy and a button
// linking to it. Skipped (idempotent) and failed platforms are omitted;
// failures are already covered by notifyFailure inside the loop. When every
// platform shares identical copy the card collapses to a single quote plus
// a button row. A no-op when no platform succeeded or when the slack sender
// is not configured.
func (s *Service) notifySuccess(ctx context.Context, pc publishCtx, results []social.PostResult) {
	if s.slack == nil {
		return
	}

	posted := make([]social.PostResult, 0, len(results))
	for _, r := range results {
		if r.Err != nil || r.Skipped || r.PostURL == "" {
			continue
		}
		posted = append(posted, r)
	}
	if len(posted) == 0 {
		return
	}

	title := "Social post published: " + kindLabel(pc.kind)
	contextLine := successContext(pc)
	fallback := title
	if pc.subject != "" {
		fallback += " - " + pc.subject
	}
	closing := slackkit.Context(fmt.Sprintf(
		"Posted to %d %s  ·  <https://godaily.dev|godaily.dev>",
		len(posted), plural(len(posted), "platform", "platforms"),
	))

	// Auto-collapse when every platform shares the same copy: one quote
	// plus a button row instead of repeating identical text per platform.
	if sameCopy(posted) {
		buttons := make([]slack.LinkButton, 0, len(posted))
		for _, r := range posted {
			buttons = append(buttons, slack.LinkButton{
				Label: "View on " + platformLabel(r.Platform),
				URL:   r.PostURL,
				Style: "primary",
			})
		}
		s.slack.MustSend(ctx, slackkit.Build(title, contextLine, fallback, slack.ColorSuccess,
			[]slackkit.Row{{Text: posted[0].Text}}, slack.ButtonRow(buttons), closing))
		return
	}

	rows := make([]slackkit.Row, 0, len(posted))
	for _, r := range posted {
		rows = append(rows, slackkit.Row{
			Heading: platformLabel(r.Platform),
			Text:    r.Text,
			Button: &slack.LinkButton{
				Label: "View on " + platformLabel(r.Platform),
				URL:   r.PostURL,
				Style: "primary",
			},
		})
	}
	s.slack.MustSend(ctx, slackkit.Build(title, contextLine, fallback, slack.ColorSuccess, rows, closing))
}

// successContext builds the issue context line for the publish card: the
// issue subject and, when known, a link to the issue in the dashboard.
func successContext(pc publishCtx) string {
	parts := make([]string, 0, 2)
	if pc.subject != "" {
		parts = append(parts, "*Issue:* "+pc.subject)
	}
	if pc.issueID != nil {
		parts = append(parts, fmt.Sprintf("<%s/issues/%d|Issue #%d>", env.DashboardURL, *pc.issueID, *pc.issueID))
	}
	return strings.Join(parts, "  ·  ")
}

// sameCopy reports whether every result carries identical post text.
func sameCopy(results []social.PostResult) bool {
	for _, r := range results[1:] {
		if r.Text != results[0].Text {
			return false
		}
	}
	return true
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
