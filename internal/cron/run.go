package cron

import (
	"context"
	"fmt"

	"github.com/ainsleyclark/godaily/internal/email"
	"github.com/ainsleyclark/godaily/internal/news"
)

type Runner interface {
	Run(RunOptions) error
}

type RunOptions struct {
	DryRun bool
}

type TODO struct {
	email *email.Client
}

func New() (*TODO, error) {
	if err := news.Validate(); err != nil {
		return nil, err
	}
	return &TODO{
		email: email.New(),
	}, nil
}

func (r TODO) Run(opts RunOptions) error {
	ctx := context.Background()

	var items []news.Item
	for _, source := range news.Sources {
		fetcher, err := news.Get(source)
		if err != nil {
			return fmt.Errorf("getting fetcher for %s: %w", source, err)
		}
		fetched, err := fetcher.Fetch(ctx)
		if err != nil {
			return fmt.Errorf("fetching %s: %w", source, err)
		}
		items = append(items, fetched...)
	}

	_ = items
	return nil
}
