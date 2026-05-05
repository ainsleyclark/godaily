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

package digest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleyclark/godaily/internal/synth"
)

// Runner is the interface for the daily news aggregation pipeline.
type Runner interface {
	Collect(ctx context.Context, opts CollectOptions) ([]news.SourceItems, error)
	SendDigest(ctx context.Context, date time.Time) error
	SendSuggestion(ctx context.Context, date time.Time) error
}

// Aggregator fetches Go news from all registered sources and optionally
// sends the digest via email.
type Aggregator struct {
	email         emailSender
	sendToAddress string
	suggester     suggester
	issues        news.IssueRepository
	items         news.ItemRepository
}

type (
	// emailSender abstracts the email client so tests can substitute a
	// fake without standing up the real Resend transport.
	emailSender interface {
		Send(ctx context.Context, req email.SendEmailRequest) error
	}
	// suggester abstracts the synth client so tests can substitute a fake
	// without hitting Anthropic. Mirrors emailSender.
	suggester interface {
		Suggest(ctx context.Context, day time.Time, sections []news.SourceItems) (synth.Suggestion, error)
	}
)

// New creates a new Aggregator, validating that all news
// sources have registered fetchers.
func New(issues news.IssueRepository, items news.ItemRepository) (*Aggregator, error) {
	if err := news.Validate(); err != nil {
		return nil, err
	}
	if (issues == nil) != (items == nil) {
		return nil, errors.New("issues and items repositories must be both set or both nil")
	}
	to := os.Getenv("EMAIL_SEND_ADDRESS")
	if to == "" {
		slog.Warn("EMAIL_SEND_ADDRESS not set, digest emails will be skipped")
	}

	// Synth is optional; without an API key we leave it nil so the
	// Aggregator still produces a digest.
	var sg suggester
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		sg = synth.New()
	} else {
		slog.Warn("ANTHROPIC_API_KEY not set, synth suggestions disabled")
	}

	return &Aggregator{
		email:         email.New(),
		sendToAddress: to,
		suggester:     sg,
		issues:        issues,
		items:         items,
	}, nil
}

func (a Aggregator) fetchSource(ctx context.Context, source news.Source) ([]news.Item, error) {
	slog.InfoContext(ctx, "fetching source", "source", source)

	fetcher, err := news.Get(source)
	if err != nil {
		return nil, fmt.Errorf("getting fetcher for %s: %w", source, err)
	}

	items, err := fetcher.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", source, err)
	}

	slog.InfoContext(ctx, "fetched from source", "source", source, "items", len(items))

	return items, nil
}

func (a Aggregator) persistIssue(ctx context.Context, issue news.Issue, sections []news.SourceItems) (news.Issue, error) {
	_, err := a.issues.FindBySlug(ctx, issue.Slug)
	switch {
	case err == nil:
		return news.Issue{}, fmt.Errorf("%w: slug %s", store.ErrAlreadyExists, issue.Slug)
	case !errors.Is(err, store.ErrNotFound):
		return news.Issue{}, fmt.Errorf("checking existing issue: %w", err)
	}

	created, err := a.issues.Create(ctx, issue)
	if err != nil {
		return news.Issue{}, fmt.Errorf("creating issue: %w", err)
	}

	var position int
	for _, section := range sections {
		for _, item := range section.Items {
			position++
			item.Source = section.Source
			if _, err = a.items.Create(ctx, created.ID, position, item); err != nil {
				return news.Issue{}, fmt.Errorf("creating news item: %w", err)
			}
		}
	}

	return created, nil
}
