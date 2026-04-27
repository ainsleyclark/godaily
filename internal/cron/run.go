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

package cron

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"time"

	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/synth"
)

// Runner is the interface for running the daily news aggregation.
type Runner interface {
	Run(ctx context.Context, opts RunOptions) ([]news.SourceItems, error)
}

// RunOptions configures a Run call.
type RunOptions struct {
	// DryRun skips sending the email digest.
	DryRun bool

	// Sources restricts the run to the given sources. If empty,
	// all registered sources (news.Sources) are used.
	Sources []news.Source

	// IncludeSynth, when true, calls the synth package after scoring
	// to draft suggested social posts and includes them in the digest.
	// A synth failure is logged but does not abort the digest.
	IncludeSynth bool
}

// Aggregator fetches Go news from all registered sources and optionally
// sends the digest via email.
type Aggregator struct {
	email         emailSender
	sendToAddress string
	suggester     suggester
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
	}, nil
}

// Run fetches Go news items published yesterday from all registered sources.
func (a Aggregator) Run(ctx context.Context, opts RunOptions) ([]news.SourceItems, error) {
	day := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour) // Yesterday
	next := day.AddDate(0, 0, 1)

	sources := opts.Sources
	if len(sources) == 0 {
		sources = news.Sources
	}

	var results []news.SourceItems
	for _, src := range sources {
		fetched, err := a.fetchSource(ctx, src)
		if err != nil {
			slog.ErrorContext(ctx, "failed to fetch source", "source", src, "err", err)
			continue
		}
		si := news.SourceItems{Source: src}

		for _, item := range fetched {
			if item.Published.IsZero() {
				slog.ErrorContext(ctx, "item has zero published date", "source", src, "title", item.Title)
				continue
			}
			if item.Published.After(day) && item.Published.Before(next) {
				si.Items = append(si.Items, item)
			}
		}

		if len(si.Items) > 0 {
			sort.SliceStable(si.Items, func(i, j int) bool {
				return si.Items[i].Score > si.Items[j].Score
			})
			results = append(results, si)
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Source.Priority() > results[j].Source.Priority()
	})

	var suggestion *synth.Suggestion
	if opts.IncludeSynth && a.suggester != nil {
		s, err := a.suggester.Suggest(ctx, day, results)
		switch {
		case errors.Is(err, synth.ErrNoItems):
			slog.InfoContext(ctx, "synth skipped: no items to summarise")
		case err != nil:
			slog.ErrorContext(ctx, "synth failed", "err", err)
		default:
			suggestion = &s
		}
	}

	if !opts.DryRun {
		if err := a.sendDigest(ctx, day, results, suggestion); err != nil {
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
