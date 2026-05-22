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
	"time"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
)

//go:generate go run go.uber.org/mock/mockgen -package=mockdigest -destination=../mocks/digest/Runner.go github.com/ainsleyclark/godaily/pkg/services/digest Runner

// Runner is the interface for the daily news aggregation pipeline.
type Runner interface {
	Collect(ctx context.Context, opts CollectOptions) ([]news.SourceItems, error)
	Build(ctx context.Context, date time.Time) error
	SendPreview(ctx context.Context, date time.Time) error
	SendDigest(ctx context.Context, date time.Time, force bool) error
	SendSuggestion(ctx context.Context, date time.Time) error
}

// slackNotifier is a minimal interface satisfied by *slack.Client, kept here
// to avoid importing the gateway package from the digest package.
type slackNotifier interface {
	MustSend(ctx context.Context, message string)
}

// Aggregator fetches Go news from all registered sources and optionally
// sends the digest via email.
type Aggregator struct {
	email             email.BatchSender
	adminEmailAddress string
	prompter          ai.Prompter
	issues            news.IssueRepository
	items             news.ItemRepository
	subscribers       news.SubscriberRepository
	slack             slackNotifier
}

// New creates a new Aggregator, validating that all news sources have
// registered fetchers. Pass a non-nil prompter to enable AI synthesis
// and suggestion; nil disables those features gracefully. Pass a non-nil
// slack to enable Slack notifications on key events; nil disables them.
func New(emailSender email.BatchSender, adminEmail string, prompter ai.Prompter, slack slackNotifier, issues news.IssueRepository, items news.ItemRepository, subscribers news.SubscriberRepository) (*Aggregator, error) {
	if err := news.Validate(); err != nil {
		return nil, err
	}
	return &Aggregator{
		email:             emailSender,
		adminEmailAddress: adminEmail,
		prompter:          prompter,
		issues:            issues,
		items:             items,
		subscribers:       subscribers,
		slack:             slack,
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
