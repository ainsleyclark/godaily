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
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	domainsocial "github.com/ainsleyclark/godaily/pkg/domain/social"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/gateway/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts/featured"
	"github.com/ainsleyclark/godaily/pkg/store"
)

//go:generate go run go.uber.org/mock/mockgen -package=mocksocialservice -destination=../../mocks/socialservice/Service.go github.com/ainsleyclark/godaily/pkg/services/social Poster

// Service publishes social media posts for both the daily featured slot
// and the Tue/Fri rotation slot.
type Service struct {
	posters    []social.Poster
	prompter   ai.Prompter
	issues     news.IssueRepository
	items      news.ItemRepository
	posts      domainsocial.PostRepository
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
		social.PlatformBluesky:  featured.Bluesky,
		social.PlatformLinkedIn: featured.LinkedIn,
		social.PlatformMastodon: featured.Mastodon,
	}
}

// New creates a new social Service. posters may be empty (nothing to post);
// the service errors if prompter, issues, items, or posts are nil.
// slackSender may be nil to disable Slack notifications. Rotation candidates
// must be wired separately via WithCandidates if Rotate will be called.
func New(
	posters []social.Poster,
	prompter ai.Prompter,
	issues news.IssueRepository,
	items news.ItemRepository,
	posts domainsocial.PostRepository,
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

// PostOptions controls a single Post invocation.
type PostOptions struct {
	// Date is the digest date — the issue slug is its UTC YYYY-MM-DD.
	Date time.Time

	// DryRun runs the full pipeline (DB read, AI calls, text generation)
	// but skips both platform HTTP and the social_posts insert.
	DryRun bool

	// Platforms optionally restricts which configured posters run. When
	// empty, every configured poster runs. Unknown platforms are ignored
	// with a log line.
	Platforms []social.Platform
}

// PostResult summarises one platform's outcome.
type PostResult struct {
	Platform social.Platform
	Kind     domainsocial.PostKind
	Text     string
	PostURL  string
	Err      error

	// Skipped is true when this platform was already posted for the same
	// idempotency key (issue or subject) on this run.
	Skipped bool
}

// Post is the entry point for the daily featured slot. It:
//  1. Loads the issue for Date (returns store.ErrNotFound if there is none).
//  2. Loads the issue's items.
//  3. Picks the day's featured item via the AI prompter.
//  4. For each enabled poster: skips if already posted today; otherwise
//     reframes the featured item for that platform, publishes it, and
//     records the result.
func (s *Service) Post(ctx context.Context, opts PostOptions) ([]PostResult, error) {
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

	featured, err := featured.Feature(ctx, s.prompter, date, rows)
	if err != nil {
		s.notifyFailure(ctx, "AI feature pick failed: "+err.Error())
		return nil, errors.Wrap(err, "feature")
	}

	slog.InfoContext(
		ctx, "Selected featured item",
		"title", featured.Title, "url", featured.URL, "tag", string(featured.Tag),
	)

	wanted := selectPosters(s.posters, opts.Platforms)
	issueID := issue.ID

	return s.publish(ctx, publishCtx{
		platforms: wanted,
		dryRun:    opts.DryRun,
		kind:      domainsocial.PostKindFeatured,
		issueID:   &issueID,
		generate: func(_ context.Context, platform social.Platform) (string, error) {
			reframe, ok := s.reframers[platform]
			if !ok {
				return "", fmt.Errorf("no reframer registered for platform %s", platform)
			}
			return reframe(ctx, s.prompter, featured)
		},
		// Featured posts use (issue_id, platform) for idempotency.
		skipIfPosted: func(ctx context.Context, platform string) (bool, error) {
			return s.posts.HasPosted(ctx, issueID, platform)
		},
	})
}

// publishCtx bundles the inputs of one publish() loop. Each rotation
// candidate produces one of these on its way through Rotate; the featured
// path constructs one inline.
type publishCtx struct {
	platforms []social.Poster
	dryRun    bool
	kind      domainsocial.PostKind
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
func (s *Service) publish(ctx context.Context, pc publishCtx) ([]PostResult, error) {
	results := make([]PostResult, 0, len(pc.platforms))
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

func (s *Service) publishOne(ctx context.Context, poster social.Poster, pc publishCtx) PostResult {
	platform := poster.Platform()
	res := PostResult{Platform: platform, Kind: pc.kind}

	if !pc.dryRun && pc.skipIfPosted != nil {
		posted, err := pc.skipIfPosted(ctx, platform.String())
		if err != nil {
			res.Err = errors.Wrap(err, "checking idempotency")
			return res
		}
		if posted {
			res.Skipped = true
			slog.InfoContext(
				ctx, "Skipping platform — already posted",
				"platform", platform, "kind", string(pc.kind),
			)
			return res
		}
	}

	text, err := pc.generate(ctx, platform)
	if err != nil {
		res.Err = errors.Wrap(err, "generate")
		return res
	}
	res.Text = text

	if pc.dryRun {
		slog.InfoContext(
			ctx, "Dry-run: skipping post + DB write",
			"platform", platform, "kind", string(pc.kind), "chars", len(text),
		)
		return res
	}

	result, err := poster.Post(ctx, text)
	if err != nil {
		res.Err = errors.Wrap(err, "poster.Post")
		return res
	}
	res.PostURL = result.PostURL

	if _, err := s.posts.Create(ctx, domainsocial.Post{
		IssueID:  pc.issueID,
		Kind:     pc.kind,
		Subject:  pc.subject,
		Platform: platform.String(),
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
		"platform", platform, "kind", string(pc.kind), "url", result.PostURL,
	)
	return res
}

// selectPosters narrows the configured posters to those requested in opts.
// When wanted is empty the full slice is returned unchanged.
func selectPosters(all []social.Poster, wanted []social.Platform) []social.Poster {
	if len(wanted) == 0 {
		return all
	}

	wantedSet := make(map[social.Platform]bool, len(wanted))
	for _, p := range wanted {
		wantedSet[p] = true
	}

	out := make([]social.Poster, 0, len(all))
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
func (s *Service) notifySuccess(ctx context.Context, pc publishCtx, results []PostResult) {
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
	case social.PlatformBluesky:
		return "Bluesky"
	case social.PlatformLinkedIn:
		return "LinkedIn"
	case social.PlatformMastodon:
		return "Mastodon"
	default:
		return string(p)
	}
}
