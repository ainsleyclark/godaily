// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/web/generate"
	"github.com/urfave/cli/v3"
)

func generateCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate static HTML files into out/ for Vercel deployment.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "out",
				Usage: "Output directory for static files",
				Value: "out",
			},
			&cli.StringFlag{
				Name:  "assets",
				Usage: "Path to compiled frontend assets",
				Value: "web/dist",
			},
			&cli.StringFlag{
				Name:  "static",
				Usage: "Path to static files copied verbatim to out/",
				Value: "web/static",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			count, _ := a.Repository.Subscribers.CountActive(ctx)
			return generate.Site(ctx, a.Repository.Issues, count, cmd.String("out"), cmd.String("static"), cmd.String("assets"))
		},
	}
}
