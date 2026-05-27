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

func previewCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "preview",
		Usage: "Send the draft digest and AI synth suggestion to the owner for early review.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "date",
				Usage: "Date of the draft to preview (YYYY-MM-DD). Defaults to today.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			date := time.Now().UTC().Truncate(24 * time.Hour)
			if raw := cmd.String("date"); raw != "" {
				d, err := time.Parse("2006-01-02", raw)
				if err != nil {
					return fmt.Errorf("invalid date %q: must be YYYY-MM-DD", raw)
				}
				date = d
			}
			return a.Service.Digest.SendPreview(ctx, date)
		},
	}
}
