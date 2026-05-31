// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/ai/anthropic"
	"github.com/ainsleyclark/godaily/pkg/domain/news"
	"github.com/ainsleyclark/godaily/pkg/services/digest/prompts"
	"github.com/urfave/cli/v3"
)

// introCmd regenerates the subject and intro for one or more stored issues
// using the current prompt and model, printing the stored values next to the
// freshly generated ones. It is read-only: issues are loaded but never written,
// and nothing is emailed, so it is safe to run against production issues.
func introCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:      "intro",
		Usage:     "Regenerate the subject and intro for stored issues and print old vs new (read-only, sends nothing).",
		ArgsUsage: "[YYYY-MM-DD ...] (defaults to yesterday)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if a.Config.AnthropicAPIKey == "" {
				return errors.New("intro requires ANTHROPIC_API_KEY")
			}
			if a.Repository.Issues == nil || a.Repository.Items == nil {
				return errors.New("intro requires persistence (TURSO_URL not set)")
			}

			slugs := cmd.Args().Slice()
			if len(slugs) == 0 {
				slugs = []string{time.Now().AddDate(0, 0, -1).Format("2006-01-02")}
			}

			prompter := anthropic.New(a.Config.AnthropicAPIKey)

			for _, slug := range slugs {
				day, err := time.Parse("2006-01-02", slug)
				if err != nil {
					return fmt.Errorf("invalid date %q: must be YYYY-MM-DD", slug)
				}

				issue, err := a.Repository.Issues.FindBySlug(ctx, slug)
				if err != nil {
					return fmt.Errorf("loading issue %s: %w", slug, err)
				}

				sections, err := introSections(ctx, a.Repository.Items, issue.ID)
				if err != nil {
					return fmt.Errorf("loading items for %s: %w", slug, err)
				}

				meta, err := prompts.Synthesise(ctx, prompter, day, sections)
				if err != nil {
					return fmt.Errorf("synthesising %s: %w", slug, err)
				}

				fmt.Printf( //nolint
					"\n=== %s (%d items) ===\n"+
						"OLD subject: %s\nNEW subject: %s\n\n"+
						"OLD intro:   %s\nNEW intro:   %s\n",
					slug, countItems(sections),
					issue.Subject, meta.Title,
					issue.Summary, meta.Intro,
				)
			}
			return nil
		},
	}
}

// introSections loads an issue's items and buckets them into per-source
// SourceItems, mirroring how the build pipeline feeds Synthesise (see
// loadSections in pkg/services/digest).
func introSections(ctx context.Context, repo news.ItemRepository, issueID int64) ([]news.SourceItems, error) {
	items, err := repo.List(ctx, news.ItemListOptions{IssueID: &issueID})
	if err != nil {
		return nil, err
	}

	order := make([]news.Source, 0)
	bySource := make(map[news.Source]*news.SourceItems)
	for _, item := range items {
		if _, ok := bySource[item.Source]; !ok {
			bySource[item.Source] = &news.SourceItems{Source: item.Source}
			order = append(order, item.Source)
		}
		bySource[item.Source].Items = append(bySource[item.Source].Items, item)
	}

	sections := make([]news.SourceItems, 0, len(bySource))
	for _, src := range order {
		sections = append(sections, *bySource[src])
	}

	sort.SliceStable(sections, func(i, j int) bool {
		return sections[i].Source.Priority() > sections[j].Source.Priority()
	})

	return sections, nil
}

// countItems totals the items across all sections.
func countItems(sections []news.SourceItems) int {
	n := 0
	for _, s := range sections {
		n += len(s.Items)
	}
	return n
}
