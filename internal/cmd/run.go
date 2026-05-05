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
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	godaily "github.com/ainsleyclark/godaily/internal"
	"github.com/ainsleyclark/godaily/internal/digest"
	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/urfave/cli/v3"
)

func runCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Collect and send the daily Go digest in one step.",
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
			sources, err := parseSources(cmd.StringSlice("source"))
			if err != nil {
				return err
			}

			dryRun := cmd.Bool("dry-run")
			date := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)

			_, raw, err := a.Aggregator.Collect(ctx, digest.CollectOptions{
				DryRun:  dryRun,
				Sources: sources,
			})
			if err != nil {
				return err
			}

			if !dryRun && len(raw) > 0 {
				if err = a.Aggregator.Send(ctx, date); err != nil {
					slog.ErrorContext(ctx, "failed to send digest", "err", err)
				}
			}

			out := cmd.String("output")
			if out == "" {
				return nil
			}

			indent, err := json.MarshalIndent(raw, "", "\t")
			if err != nil {
				return err
			}

			if err = os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
				return err
			}

			return os.WriteFile(out, indent, 0o600)
		},
	}
}

// parseSources validates a slice of raw source name strings against the
// registered sources list and returns the typed slice.
func parseSources(raw []string) ([]news.Source, error) {
	known := make(map[news.Source]struct{}, len(news.Sources))
	for _, s := range news.Sources {
		known[s] = struct{}{}
	}
	sources := make([]news.Source, 0, len(raw))
	for _, name := range raw {
		s := news.Source(name)
		if _, ok := known[s]; !ok {
			return nil, fmt.Errorf("unknown source %q (run `godaily sources` for the list)", name)
		}
		sources = append(sources, s)
	}
	return sources, nil
}
