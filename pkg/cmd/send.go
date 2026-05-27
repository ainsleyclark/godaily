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

func sendCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "send",
		Usage: "Send the stored draft digest via email.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "date",
				Usage: "Date of the draft to send (YYYY-MM-DD). Defaults to yesterday.",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Send even if the digest is not in draft status.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			date := time.Now()
			if raw := cmd.String("date"); raw != "" {
				d, err := time.Parse("2006-01-02", raw)
				if err != nil {
					return fmt.Errorf("invalid date %q: must be YYYY-MM-DD", raw)
				}
				date = d
			}
			if err := a.Service.Digest.SendDigest(ctx, date, cmd.Bool("force")); err != nil {
				a.Slack.MustSend(ctx, "Send digest failed: "+err.Error())
				return err
			}
			return nil
		},
	}
}
