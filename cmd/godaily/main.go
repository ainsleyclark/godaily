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
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/ainsleyclark/godaily/internal/cron"
	"github.com/ainsleyclark/godaily/internal/news"
	_ "github.com/ainsleyclark/godaily/internal/source"
	"github.com/urfave/cli/v3"
)

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
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				runner, err := cron.New()
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
					DryRun:  cmd.Bool("dry-run"),
					Sources: sources,
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

				if err = os.MkdirAll(filepath.Dir(out), os.ModePerm); err != nil {
					return err
				}

				return os.WriteFile(out, indent, 0o644)
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
