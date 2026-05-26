// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
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
			now := time.Now().UTC()
			today := now.Truncate(24 * time.Hour)
			date := today.AddDate(0, 0, -1)
			if now.Weekday() == time.Monday {
				date = today.AddDate(0, 0, -2)
			}

			raw, err := a.Runner.Collect(ctx, digest.CollectOptions{
				DryRun:  dryRun,
				Sources: sources,
			})
			if err != nil {
				return err
			}

			if !dryRun && len(raw.Sources) > 0 {
				if err = a.Runner.SendDigest(ctx, date, false); err != nil {
					return err
				}
			}

			out := cmd.String("output")
			if out == "" {
				return nil
			}

			indent := prettyJSON(raw)

			if err = os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
				return err
			}

			return os.WriteFile(out, indent, 0o600)
		},
	}
}
