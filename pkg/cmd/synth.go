// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"time"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/urfave/cli/v3"
)

func synthCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "synth",
		Usage: "Generate an AI post suggestion from the stored digest and email it to the owner.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "date",
				Usage: "Date of the digest to use (YYYY-MM-DD). Defaults to yesterday.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			date := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
			if raw := cmd.String("date"); raw != "" {
				d, err := time.Parse("2006-01-02", raw)
				if err != nil {
					return fmt.Errorf("invalid date %q: must be YYYY-MM-DD", raw)
				}
				date = d
			}
			return a.Runner.SendSuggestion(ctx, date)
		},
	}
}
