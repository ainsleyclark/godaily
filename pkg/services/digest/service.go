// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package digest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ainsleyclark/godaily/pkg/ai"
	"github.com/ainsleyclark/godaily/pkg/domain/audience"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/gateway/email"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
)

// Service fetches Go news from all registered sources and optionally
// sends the digest via email.
type Service struct {
	email             email.BatchSender
	adminEmailAddress string
	prompter          ai.Prompter
	issues            digest.IssueRepository
	items             news.ItemRepository
	subscribers       audience.SubscriberRepository
	slack             slack.Sender
}

var _ digest.Service = (*Service)(nil)

// New creates a new Service, validating that all news sources have
// registered fetchers.
func New(
	emailSender email.BatchSender,
	adminEmail string,
	prompter ai.Prompter,
	slack slack.Sender,
	issues digest.IssueRepository,
	items news.ItemRepository,
	subscribers audience.SubscriberRepository,
) (*Service, error) {
	if news.HasSources() {
		if err := news.Validate(); err != nil {
			return nil, err
		}
	}
	return &Service{
		email:             emailSender,
		adminEmailAddress: adminEmail,
		prompter:          prompter,
		issues:            issues,
		items:             items,
		subscribers:       subscribers,
		slack:             slack,
	}, nil
}

func (s Service) fetchSource(ctx context.Context, source news.Source) ([]news.Item, error) {
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
