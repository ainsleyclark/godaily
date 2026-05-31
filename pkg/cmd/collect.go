// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"os"
	"path/filepath"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/domain/digest"
	"github.com/ainsleyclark/godaily/pkg/gateway/slack"
	"github.com/urfave/cli/v3"
)

func collectCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "collect",
		Usage: "Fetch Go news from all sources and store the digest as a draft.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Skip persisting the digest; only gather and return raw items",
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

			raw, err := a.Service.Digest.Collect(ctx, digest.CollectOptions{
				DryRun:  cmd.Bool("dry-run"),
				Sources: sources,
			})
			if err != nil {
				a.Slack.MustSend(ctx, slack.Error("Collect failed (CLI)", err))
				return err
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
