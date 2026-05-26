// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/web/server"
	"github.com/urfave/cli/v3"
)

func serveCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start the HTTP web server.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Usage:   "Port to listen on",
				Value:   "3000",
				Sources: cli.EnvVars("PORT"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return server.Start(a, cmd.String("port"))
		},
	}
}
