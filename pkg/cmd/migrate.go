// Copyright (c) 2026 godaily (Ainsley Clark) All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"

	godaily "github.com/ainsleyclark/godaily/pkg"
	"github.com/ainsleyclark/godaily/pkg/db"
	"github.com/urfave/cli/v3"
)

func migrateCmd(a *godaily.App) *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Manage database migrations",
		Commands: []*cli.Command{
			{
				Name:  "up",
				Usage: "Apply pending database migrations",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return db.Up(ctx, a.DB)
				},
			},
			{
				Name:  "down",
				Usage: "Roll back the most recent migration",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return db.Down(ctx, a.DB)
				},
			},
		},
	}
}
