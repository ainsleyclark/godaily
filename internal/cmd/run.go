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
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ainsleyclark/godaily/internal/cron"
	"github.com/ainsleyclark/godaily/internal/db"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/store/issues"
	"github.com/ainsleyclark/godaily/internal/store/items"
	"github.com/urfave/cli/v3"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Gather all news from sources.",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "dry-run",
			Usage: "Skip sending the digest email",
		},
		&cli.StringFlag{
			Name:  "output",
			Usage: "Write aggregated items as JSON to this path (skipped if empty)",
		},
		&cli.StringSliceFlag{
			Name:  "source",
			Usage: "Only run the named sources (repeatable). Defaults to all.",
		},
		&cli.BoolFlag{
			Name:  "synth",
			Value: true,
			Usage: "Also generate suggested social posts via Anthropic and include them in the digest",
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

		// Wrap the concrete *Store values into typed-nil-safe interface
		// variables: a nil *issues.Store passed as news.IssueRepository
		// is *not* == nil, which would defeat the nil-check inside cron.
		var (
			issueRepo news.IssueRepository
			itemRepo  news.ItemRepository
		)
		if issueStore != nil {
			issueRepo = issueStore
			itemRepo = itemStore
		}

		runner, err := cron.New(issueRepo, itemRepo)
		if err != nil {
			return err
		}

		raw := cmd.StringSlice("source")
		known := make(map[news.Source]struct{}, len(news.Sources))
		for _, s := range news.Sources {
			known[s] = struct{}{}
		}
		sources := make([]news.Source, 0, len(raw))
		for _, name := range raw {
			s := news.Source(name)
			if _, ok := known[s]; !ok {
				return fmt.Errorf("unknown source %q (run `godaily sources` for the list)", name)
			}
			sources = append(sources, s)
		}

		items, err := runner.Run(ctx, cron.RunOptions{
			DryRun:       cmd.Bool("dry-run"),
			Sources:      sources,
			IncludeSynth: cmd.Bool("synth"),
		})
		if err != nil {
			return err
		}

		out := cmd.String("output")
		if out == "" {
			return nil
		}

		indent, err := json.MarshalIndent(items, "", "\t")
		if err != nil {
			return err
		}

		if err = os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
			return err
		}

		return os.WriteFile(out, indent, 0o600)
	},
}

// openStores connects to the configured database and returns the issue
// and item stores, plus the underlying *sql.DB so the caller can close it.
//
// If TURSO_URL is unset the function returns (nil, nil, nil, nil) so the
// rest of the CLI can run without a database — useful for ad-hoc
// fetch/synth commands during development.
func openStores(ctx context.Context) (*issues.Store, *items.Store, *sql.DB, error) {
	url := os.Getenv("TURSO_URL")
	if url == "" {
		slog.WarnContext(ctx, "TURSO_URL not set, persistence disabled")
		return nil, nil, nil, nil
	}

	conn, err := db.New(ctx, url, os.Getenv("TURSO_AUTH_TOKEN"))
	if err != nil {
		return nil, nil, nil, err
	}

	if err = db.Up(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("running migrations: %w", err)
	}

	return issues.New(conn), items.New(conn), conn, nil
}
