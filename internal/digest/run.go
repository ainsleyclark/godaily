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
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
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
	email             emailSender
	adminEmailAddress string
	suggester         suggester
	issues            news.IssueRepository
	items             news.ItemRepository
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
	adminEmailAddress := os.Getenv("EMAIL_SEND_ADDRESS")
	if adminEmailAddress == "" {
		adminEmailAddress = "hello@ainsley.dev"
	}
	return &Aggregator{
		email:             email.New(),
		adminEmailAddress: adminEmailAddress,
		suggester:         synth.New(),
		issues:            issues,
		items:             items,
	}, nil
}

func (a Aggregator) fetchSource(ctx context.Context, source news.Source) ([]news.Item, error) {
	slog.InfoContext(ctx, "Fetching source", "source", source)

	fetcher, err := news.Get(source)
	if err != nil {
		return nil, fmt.Errorf("getting fetcher for %s: %w", source, err)
	}

	items, err := fetcher.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", source, err)
	}

	slog.InfoContext(ctx, "Fetched from source", "source", source, "items", len(items))

	return items, nil
}
