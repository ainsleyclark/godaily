package cron

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
)

// Runner is the interface for running the daily news aggregation.
type Runner interface {
	Run(ctx context.Context, opts RunOptions) ([]news.SourceItems, error)
}

// RunOptions configures a Run call.
type RunOptions struct {
	// DryRun skips sending the email digest.
	DryRun bool
}

// Aggregator fetches Go news from all registered sources and optionally
// sends the digest via email.
type Aggregator struct {
	email         *email.Client
	sendToAddress string
}

// New creates a new Aggregator, validating that all news sources have
// registered fetchers before returning.
func New() (*Aggregator, error) {
	if err := news.Validate(); err != nil {
		return nil, err
	}
	to := os.Getenv("EMAIL_SEND_ADDRESS")
	if to == "" {
		slog.Warn("EMAIL_SEND_ADDRESS not set, digest emails will be skipped")
	}
	return &Aggregator{
		email:         email.New(),
		sendToAddress: to,
	}, nil
}

// Run fetches Go news items published yesterday from all registered sources.
func (a Aggregator) Run(ctx context.Context, opts RunOptions) ([]news.SourceItems, error) {
	day := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour) // Yesterday
	next := day.AddDate(0, 0, 1)

	var results []news.SourceItems
	for _, source := range news.Sources {
		fetched, err := a.fetchSource(ctx, source)
		if err != nil {
			slog.ErrorContext(ctx, "failed to fetch source", "source", source, "err", err)
			continue
		}
		si := news.SourceItems{Source: source}

		for _, item := range fetched {
			if item.Published.IsZero() {
				slog.ErrorContext(ctx, "item has zero published date", "source", source, "title", item.Title)
				continue
			}
			if item.Published.After(day) && item.Published.Before(next) {
				si.Items = append(si.Items, item)
			}
		}

		if len(si.Items) > 0 {
			results = append(results, si)
		}
	}

	if !opts.DryRun {
		if err := a.sendDigest(ctx, day, results); err != nil {
			slog.ErrorContext(ctx, "failed to send digest email", "err", err)
		}
	}

	return results, nil
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
