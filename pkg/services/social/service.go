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

// Package social orchestrates social media posting for a digest issue:
// picks the day's featured news item via the AI prompter, reframes it for
// each enabled platform, publishes to each platform's Poster, and records
// the result for idempotency and audit.
package social

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/ainsleyclark/godaily/pkg/gateway/social"
	"github.com/ainsleyclark/godaily/pkg/services/social/prompts"
	"github.com/ainsleyclark/godaily/pkg/store"
)

//go:generate go run go.uber.org/mock/mockgen -package=mocksocialservice -destination=../../mocks/socialservice/Service.go github.com/ainsleyclark/godaily/pkg/services/social Poster

// Service publishes social media posts for the day's digest.
type Service struct {
	posters   []social.Poster
	prompter  ai.Prompter
	issues    news.IssueRepository
	items     news.ItemRepository
	posts     news.SocialPostRepository
	slack     slack.Sender
	reframers map[social.Platform]reframer
}

// reframer reframes a featured item for one platform. Function-typed so
// tests can inject deterministic text without going through the AI.
type reframer func(ctx context.Context, p ai.Prompter, f prompts.Featured) (string, error)

// defaultReframers maps each Platform to its production reframing prompt.
func defaultReframers() map[social.Platform]reframer {
	return map[social.Platform]reframer{
		social.PlatformBluesky:  prompts.Bluesky,
		social.PlatformLinkedIn: prompts.LinkedIn,
		social.PlatformMastodon: prompts.Mastodon,
	}
}

// New creates a new social Service. posters may be empty (nothing to post);
// the service errors if prompter, issues, items, or posts are nil.
// slackSender may be nil to disable Slack notifications.
func New(
	posters []social.Poster,
	prompter ai.Prompter,
	issues news.IssueRepository,
	items news.ItemRepository,
	posts news.SocialPostRepository,
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
	Text     string
	PostURL  string
	Err      error

	// Skipped is true when this platform was already posted today.
	Skipped bool
}

// Post is the main entry point. It:
//  1. Loads the issue for Date (returns store.ErrNotFound if there is none).
//  2. Loads the issue's items.
//  3. Picks the day's featured item via the AI prompter.
//  4. For each enabled poster: skips if already posted today; otherwise
//     reframes the featured item for that platform, publishes it, and
//     records the result.
//
// On a per-platform error the loop continues and the caller receives a
// joined error. The slack notifier is pinged on any failure.
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

	featured, err := prompts.Feature(ctx, s.prompter, date, rows)
	if err != nil {
		s.notifyFailure(ctx, "AI feature pick failed: "+err.Error())
		return nil, errors.Wrap(err, "feature")
	}

	slog.InfoContext(
		ctx, "Selected featured item",
		"title", featured.Title, "url", featured.URL, "tag", string(featured.Tag),
	)

	wanted := selectPosters(s.posters, opts.Platforms)

	results := make([]PostResult, 0, len(wanted))
	var errs []error

	for _, poster := range wanted {
		res := s.postOne(ctx, issue.ID, poster, featured, opts.DryRun)
		results = append(results, res)
		if res.Err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", res.Platform, res.Err))
			s.notifyFailure(ctx, fmt.Sprintf("Social post to %s failed: %s", res.Platform, res.Err))
		}
	}

	if len(errs) > 0 {
		return results, stderrors.Join(errs...)
	}
	return results, nil
}

// postOne handles one platform end-to-end. It is internal so the loop in
// Post can keep tallying results even when a platform fails.
func (s *Service) postOne(
	ctx context.Context,
	issueID int64,
	poster social.Poster,
	featured prompts.Featured,
	dryRun bool,
) PostResult {
	platform := poster.Platform()
	res := PostResult{Platform: platform}

	if !dryRun {
		posted, err := s.posts.HasPosted(ctx, issueID, platform.String())
		if err != nil {
			res.Err = errors.Wrap(err, "checking HasPosted")
			return res
		}
		if posted {
			res.Skipped = true
			slog.InfoContext(
				ctx, "Skipping platform — already posted today",
				"platform", platform, "issue", issueID,
			)
			return res
		}
	}

	reframe, ok := s.reframers[platform]
	if !ok {
		res.Err = fmt.Errorf("no reframer registered for platform %s", platform)
		return res
	}

	text, err := reframe(ctx, s.prompter, featured)
	if err != nil {
		res.Err = errors.Wrap(err, "reframe")
		return res
	}
	res.Text = text

	if dryRun {
		slog.InfoContext(
			ctx, "Dry-run: skipping post + DB write",
			"platform", platform, "chars", len(text),
		)
		return res
	}

	result, err := poster.Post(ctx, text)
	if err != nil {
		res.Err = errors.Wrap(err, "poster.Post")
		return res
	}
	res.PostURL = result.PostURL

	if _, err := s.posts.Create(ctx, news.SocialPost{
		IssueID:  issueID,
		Platform: platform.String(),
		Text:     text,
		PostURL:  result.PostURL,
	}); err != nil {
		// The platform post already succeeded — failing the DB write
		// is bad but not fatal (we'll notify Slack so we can backfill).
		res.Err = errors.Wrap(err, "recording social_post")
		return res
	}

	slog.InfoContext(
		ctx, "Social post published",
		"platform", platform, "url", result.PostURL, "issue", issueID,
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
