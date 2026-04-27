package cron

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
)

// Runner is the interface for running the daily news aggregation.
type Runner interface {
	Run(ctx context.Context, opts RunOptions) ([]news.Item, error)
}

// RunOptions configures a Run call.
type RunOptions struct {
	// DryRun skips sending the email digest.
	DryRun bool
}

// Aggregator fetches Go news from all registered sources and optionally
// sends the digest via email.
type Aggregator struct {
	email *email.Client
}

// New creates a new Aggregator, validating that all news sources have
// registered fetchers before returning.
func New() (*Aggregator, error) {
	if err := news.Validate(); err != nil {
		return nil, err
	}
	return &Aggregator{
		email: email.New(),
	}, nil
}

// Run fetches Go news items published yesterday from all registered sources.
// Errors from individual sources are logged and skipped rather than aborting
// the run.
func (a Aggregator) Run(ctx context.Context, opts RunOptions) ([]news.Item, error) {
	day := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	next := day.AddDate(0, 0, 1)

	var items []news.Item
	for _, source := range news.Sources {
		fetched, err := a.fetchSource(ctx, source)
		if err != nil {
			slog.ErrorContext(ctx, "failed to fetch source", "source", source, "err", err)
			continue
		}
		for _, item := range fetched {
			if !item.Published.IsZero() && item.Published.After(day) && item.Published.Before(next) {
				items = append(items, item)
			}
		}
	}

	return items, nil
}

func (a Aggregator) fetchSource(ctx context.Context, source news.Source) ([]news.Item, error) {
	fetcher, err := news.Get(source)
	if err != nil {
		return nil, fmt.Errorf("getting fetcher for %s: %w", source, err)
	}
	items, err := fetcher.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", source, err)
	}
	return items, nil
}
