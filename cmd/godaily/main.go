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

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"github.com/ainsleyclark/godaily/internal/cron"
	"github.com/ainsleyclark/godaily/internal/db"
	"github.com/ainsleyclark/godaily/internal/news"
	_ "github.com/ainsleyclark/godaily/internal/source"
	"github.com/ainsleyclark/godaily/internal/store"
	"github.com/ainsleyclark/godaily/internal/synth"
	"github.com/urfave/cli/v3"
)

// openStore connects to the configured database and returns a *store.Store.
//
// If TURSO_URL is unset the function returns (nil, nil) so the rest of the
// CLI can run without a database — useful for ad-hoc fetch/synth commands
// during development.
func openStore(ctx context.Context) (*store.Store, *sql.DB, error) {
	url := os.Getenv("TURSO_URL")
	if url == "" {
		slog.WarnContext(ctx, "TURSO_URL not set, persistence disabled")
		return nil, nil, nil
	}
	conn, err := db.New(ctx, url, os.Getenv("TURSO_AUTH_TOKEN"))
	if err != nil {
		return nil, nil, err
	}
	return store.NewStore(conn), conn, nil
}

var cmd = &cli.Command{
	Name:  "godaily",
	Usage: "Daily Go news, straight to your inbox",
	Commands: []*cli.Command{
		{
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
				st, conn, err := openStore(ctx)
				if err != nil {
					return fmt.Errorf("opening database: %w", err)
				}
				if conn != nil {
					defer conn.Close()
				}

				// st may be nil — cron.New tolerates a nil persister and
				// simply skips archival when no DB is configured.
				var p cron.Persister
				if st != nil {
					p = st
				}
				runner, err := cron.New(p)
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
		},
		{
			Name:  "sources",
			Usage: "Lists registered source names",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				for _, name := range news.Sources {
					fmt.Println(name) //nolint
				}
				return nil
			},
		},
		{
			Name:  "synth",
			Usage: "Suggest a tweet and LinkedIn post from a scored news JSON file",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "input",
					Usage:    "Path to a scored news JSON file (the output of `godaily run --output`)",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "output",
					Usage: "Write suggestion to this path (otherwise prints to stdout)",
				},
				&cli.StringFlag{
					Name:  "format",
					Value: "md",
					Usage: "Output format: md or json",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				raw, err := os.ReadFile(cmd.String("input"))
				if err != nil {
					return fmt.Errorf("read input: %w", err)
				}

				var sections []news.SourceItems
				if err := json.Unmarshal(raw, &sections); err != nil {
					return fmt.Errorf("parse input: %w", err)
				}

				sug, err := synth.New().Suggest(ctx, time.Now().AddDate(0, 0, -1).Truncate(24*time.Hour), sections)
				if err != nil {
					return err
				}

				var rendered []byte
				switch cmd.String("format") {
				case "json":
					rendered, err = sug.JSON()
					if err != nil {
						return err
					}
				case "md", "":
					rendered = []byte(sug.Markdown())
				default:
					return fmt.Errorf("unknown format %q (want md or json)", cmd.String("format"))
				}

				out := cmd.String("output")
				if out == "" {
					fmt.Println(string(rendered)) //nolint
					return nil
				}
				if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
					return err
				}
				return os.WriteFile(out, rendered, 0o600)
			},
		},
		{
			Name:  "migrate",
			Usage: "Apply pending database migrations",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "down",
					Usage: "Roll back the most recent migration instead of applying pending ones",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				url := os.Getenv("TURSO_URL")
				if url == "" {
					return fmt.Errorf("TURSO_URL is required for the migrate command")
				}
				conn, err := db.New(ctx, url, os.Getenv("TURSO_AUTH_TOKEN"))
				if err != nil {
					return fmt.Errorf("opening database: %w", err)
				}
				defer conn.Close()

				if cmd.Bool("down") {
					return db.Down(ctx, conn)
				}
				return db.Migrate(ctx, conn)
			},
		},
		{
			Name: "fetch",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "provider",
					Usage: "Provider of source information",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				fetcher, err := news.Get(news.Source(cmd.String("provider")))
				if err != nil {
					return err
				}

				items, err := fetcher.Fetch(ctx)
				if err != nil {
					return err
				}

				indent, err := json.MarshalIndent(items, "", "  ")
				if err != nil {
					return err
				}

				fmt.Println(string(indent)) //nolint
				return nil
			},
		},
	},
}

func main() {
	if err := godotenv.Load(); err != nil {
		slog.ErrorContext(context.Background(), "error loading .env file")
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
