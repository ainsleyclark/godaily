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

package cmd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ainsleyclark/godaily/internal/digest"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/urfave/cli/v3"
)

var sendCmd = &cli.Command{
	Name:  "send",
	Usage: "Send the stored draft digest via email.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "date",
			Usage: "Date of the draft to send (YYYY-MM-DD). Defaults to yesterday.",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		issueStore, itemStore, conn, err := openStores(ctx)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		if conn != nil {
			defer conn.Close()
		}
		if issueStore == nil {
			return fmt.Errorf("TURSO_URL must be set to send a digest")
		}

		date := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
		if raw := cmd.String("date"); raw != "" {
			date, err = time.Parse("2006-01-02", raw)
			if err != nil {
				return fmt.Errorf("invalid date %q: must be YYYY-MM-DD", raw)
			}
		}
		slug := date.Format("2006-01-02")

		issue, err := issueStore.FindBySlug(ctx, slug)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fmt.Errorf("no digest found for %s — run `godaily collect` first", slug)
			}
			return fmt.Errorf("loading digest: %w", err)
		}
		if issue.Status != news.IssueStatusDraft {
			return fmt.Errorf("digest for %s has status %q, expected %q", slug, issue.Status, news.IssueStatusDraft)
		}

		// Load stored items and reconstruct sections so the synth suggestion
		// can be generated at send time without being persisted in the DB.
		sections, err := sectionsFromDB(ctx, itemStore, issue.ID)
		if err != nil {
			return fmt.Errorf("loading items: %w", err)
		}

		runner, err := digest.New(issueStore, itemStore)
		if err != nil {
			return err
		}

		return runner.Send(ctx, issue, sections)
	},
}

// sectionsFromDB loads stored items for an issue and groups them into
// SourceItems slices sorted by source priority, matching the shape
// produced by Collect.
func sectionsFromDB(ctx context.Context, repo news.ItemRepository, issueID int64) ([]news.SourceItems, error) {
	items, err := repo.ListByIssue(ctx, issueID)
	if err != nil {
		return nil, err
	}

	// Preserve position order (score-desc) within each source group.
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
